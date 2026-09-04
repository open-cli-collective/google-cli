package architecture

import (
	"bytes"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrintedDTOTextIsSanitized(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	textFields := dtoTextFields(t, root)
	wrappedSites := 0

	for _, kind := range []string{"cmd", "rwcmd"} {
		for _, pkg := range packageDirs(t, kind) {
			dir := filepath.Join(root, "internal", kind, pkg)
			files, paths := parseNonTestFiles(t, dir), nonTestGoFiles(t, dir)
			for i, file := range files {
				ast.Inspect(file, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok || !isOutputCall(call) {
						return true
					}
					wrapped := false
					for _, arg := range call.Args {
						inspectPrintedArgument(arg, textFields, false, func(sel *ast.SelectorExpr, sanitized bool) {
							if sanitized {
								wrapped = true
								return
							}
							t.Errorf("%s:%d: printed DTO field %s must be wrapped in sanitize.Output or sanitize.Filename", relativePath(root, paths[i]), sourceLine(t, paths[i], sel.Pos()), sel.Sel.Name)
						})
					}
					if wrapped {
						wrappedSites++
					}
					return true
				})
			}
		}
	}

	if wrappedSites < 20 {
		t.Fatalf("saw %d sanitized DTO print sites, want at least 20", wrappedSites)
	}
}

func dtoTextFields(t *testing.T, root string) map[string]bool {
	t.Helper()
	fields := map[string]bool{}
	for _, pkg := range packageDirs(t, "api") {
		for _, file := range parseNonTestFiles(t, filepath.Join(root, "internal", "api", pkg)) {
			ast.Inspect(file, func(node ast.Node) bool {
				structType, ok := node.(*ast.StructType)
				if !ok {
					return true
				}
				for _, field := range structType.Fields.List {
					if !isStringOrStringSlice(field.Type) {
						continue
					}
					for _, name := range field.Names {
						if name.IsExported() {
							fields[name.Name] = true
						}
					}
				}
				return true
			})
		}
	}

	// These fields are identifiers or machine-readable values, not prose.
	for _, name := range []string{"ID", "ResourceName", "ThreadID", "MessageID", "MimeType", "Type", "Status", "ETag", "TimeZone", "Date", "DateTime"} {
		delete(fields, name)
	}
	for name := range fields {
		if strings.HasSuffix(name, "ID") || strings.HasSuffix(name, "IDs") || strings.HasSuffix(name, "URL") || strings.HasSuffix(name, "Link") {
			delete(fields, name)
		}
	}
	return fields
}

func isStringOrStringSlice(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "string"
	}
	array, ok := expr.(*ast.ArrayType)
	if !ok || array.Len != nil {
		return false
	}
	ident, ok := array.Elt.(*ast.Ident)
	return ok && ident.Name == "string"
}

func isOutputCall(call *ast.CallExpr) bool {
	if ident, ok := call.Fun.(*ast.Ident); ok {
		return ident.Name == "append" && len(call.Args) > 0 && isIdent(call.Args[0], "rows")
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	methods := map[string]bool{"Printf": true, "Println": true, "Success": true, "Info": true, "Error": true}
	if methods[sel.Sel.Name] {
		return true
	}
	fmtFunctions := map[string]bool{"Print": true, "Fprint": true, "Fprintf": true, "Fprintln": true, "Sprintf": true, "Sprint": true}
	return fmtFunctions[sel.Sel.Name] && isIdent(sel.X, "fmt")
}

func inspectPrintedArgument(expr ast.Expr, fields map[string]bool, sanitized bool, visit func(*ast.SelectorExpr, bool)) {
	ast.Inspect(expr, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && isSanitizerCall(call) {
			for _, arg := range call.Args {
				inspectPrintedArgument(arg, fields, true, visit)
			}
			return false
		}
		if sel, ok := node.(*ast.SelectorExpr); ok && fields[sel.Sel.Name] {
			visit(sel, sanitized)
		}
		return true
	})
}

func isSanitizerCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && isIdent(sel.X, "sanitize") && (sel.Sel.Name == "Output" || sel.Sel.Name == "Filename")
}

func isIdent(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

func nonTestGoFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var paths []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			paths = append(paths, filepath.Join(dir, name))
		}
	}
	return paths
}

func sourceLine(t *testing.T, path string, pos token.Pos) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	offset := int(pos) - 1
	if offset < 0 || offset > len(data) {
		return 0
	}
	return bytes.Count(data[:offset], []byte("\n")) + 1
}

func relativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
