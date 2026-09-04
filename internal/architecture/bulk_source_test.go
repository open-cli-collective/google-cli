package architecture

import (
	"go/ast"
	"os"
	"strconv"
	"strings"
	"testing"
)

// minDeclaringFiles is a sanity floor: the repo has well over ten files that
// take bulk IDs, so a detector that matches fewer has stopped seeing them.
const minDeclaringFiles = 10

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
	if declaringFiles < minDeclaringFiles {
		t.Fatalf("found %d files declaring bulk ID sources; want at least %d", declaringFiles, minDeclaringFiles)
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
		if !nameOK || !helpOK {
			return true
		}
		// A bulk ID source is a --stdin or --query flag whose help says it
		// carries IDs or resource names. The same flag names also appear on
		// mail draft (body from stdin) and mail filter (filter criteria),
		// which are not ID sources.
		isBulkFlagName := name == "stdin" || name == "query"
		helpMentionsIDs := strings.Contains(help, "ID") || strings.Contains(help, "resource name")
		if isBulkFlagName && helpMentionsIDs {
			flags[name] = true
		}
		return true
	})
	return flags
}

// resolverFunctions returns the package's functions that reach
// bulk.ResolveIDs. It iterates to a fixed point because a leaf may call a
// helper (such as the drive package's resolveFileIDs) that calls another
// helper before the resolver; a single pass would only see direct callers.
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
