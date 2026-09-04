package architecture

import (
	"go/ast"
	"go/parser"
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
			for _, source := range nonTestSources(t, dir) {
				ast.Inspect(source.file, func(node ast.Node) bool {
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
							t.Errorf("%s:%d: printed DTO text %s must be wrapped in sanitize.Output or sanitize.Filename", relativePath(root, source.path), source.fset.Position(sel.Pos()).Line, sel.Sel.Name)
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

// dtoTextFields returns the names of DTO text sources: exported string and
// []string struct fields in internal/api, plus the argument-less Get* string
// methods declared on those DTOs (GetDisplayName and friends), so text reached
// through a getter is held to the same rule as a field. Formatting helpers
// such as FormatTimeRange are not getters and render only dates.
func dtoTextFields(t *testing.T, root string) map[string]bool {
	t.Helper()
	fields := map[string]bool{}
	for _, pkg := range packageDirs(t, "api") {
		for _, file := range parseNonTestFiles(t, filepath.Join(root, "internal", "api", pkg)) {
			ast.Inspect(file, func(node ast.Node) bool {
				if fn, ok := node.(*ast.FuncDecl); ok {
					if fn.Recv != nil && strings.HasPrefix(fn.Name.Name, "Get") && isStringGetter(fn.Type) {
						fields[fn.Name.Name] = true
					}
					return false
				}
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

func isStringGetter(fn *ast.FuncType) bool {
	if fn.Params != nil && len(fn.Params.List) > 0 {
		return false
	}
	if fn.Results == nil || len(fn.Results.List) != 1 {
		return false
	}
	return isIdent(fn.Results.List[0].Type, "string")
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

// parsedSource keeps a parsed file together with the path and file set needed
// to report positions in it.
type parsedSource struct {
	path string
	fset *token.FileSet
	file *ast.File
}

func nonTestSources(t *testing.T, dir string) []parsedSource {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var sources []parsedSource
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		sources = append(sources, parsedSource{path: path, fset: fset, file: file})
	}
	return sources
}

func relativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}
