package architecture

import (
	"go/ast"
	"os"
	"strconv"
	"strings"
	"testing"
)

func TestBulkLeavesRouteThroughResolver(t *testing.T) {
	t.Parallel()
	declaringFiles := 0
	for _, pkg := range commandPackages(t) {
		files := parseNonTestFiles(t, pkg.dir)
		names := nonTestFileNames(t, pkg.dir)
		resolvers := resolverFunctions(files)
		for i, file := range files {
			flags := bulkSourceFlags(file)
			if len(flags) == 0 {
				continue
			}
			declaringFiles++
			for flag := range flags {
				if !callsResolver(file, resolvers) {
					t.Errorf("internal/%s/%s declares --%s but does not route through bulk.ResolveIDs", pkg.name, names[i], flag)
				}
			}
		}
	}
	if declaringFiles < 10 {
		t.Fatalf("found %d files declaring bulk ID sources; want at least 10", declaringFiles)
	}
}

func nonTestFileNames(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	return names
}

func bulkSourceFlags(file *ast.File) map[string]bool {
	flags := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || len(call.Args) < 3 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		flagsCall, ok := sel.X.(*ast.CallExpr)
		if !ok || !isSelector(flagsCall.Fun, "", "Flags") {
			return true
		}
		name, nameOK := stringLiteral(call.Args[1])
		help, helpOK := stringLiteral(call.Args[len(call.Args)-1])
		if nameOK && helpOK && (name == "stdin" || name == "query") && (strings.Contains(help, "ID") || strings.Contains(help, "resource name")) {
			flags[name] = true
		}
		return true
	})
	return flags
}

func resolverFunctions(files []*ast.File) map[string]bool {
	resolvers := map[string]bool{}
	for changed := true; changed; {
		changed = false
		for _, file := range files {
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil || resolvers[fn.Name.Name] {
					continue
				}
				if callsResolver(fn.Body, resolvers) {
					resolvers[fn.Name.Name] = true
					changed = true
				}
			}
		}
	}
	return resolvers
}

func callsResolver(node ast.Node, resolvers map[string]bool) bool {
	found := false
	ast.Inspect(node, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if isSelector(call.Fun, "bulk", "ResolveIDs") {
			found = true
		} else if ident, ok := call.Fun.(*ast.Ident); ok && resolvers[ident.Name] {
			found = true
		}
		return !found
	})
	return found
}

func isSelector(expr ast.Expr, receiver, name string) bool {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != name {
		return false
	}
	if receiver == "" {
		return true
	}
	ident, ok := sel.X.(*ast.Ident)
	return ok && ident.Name == receiver
}

func stringLiteral(expr ast.Expr) (string, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}
