package architecture

import (
	"bytes"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	groapp "github.com/open-cli-collective/google-cli/internal/app/gro"
	grwapp "github.com/open-cli-collective/google-cli/internal/app/grw"
	calcmd "github.com/open-cli-collective/google-cli/internal/cmd/calendar"
	contactscmd "github.com/open-cli-collective/google-cli/internal/cmd/contacts"
	drivecmd "github.com/open-cli-collective/google-cli/internal/cmd/drive"
	mailcmd "github.com/open-cli-collective/google-cli/internal/cmd/mail"
	mecmd "github.com/open-cli-collective/google-cli/internal/cmd/me"
	rwmailcmd "github.com/open-cli-collective/google-cli/internal/rwcmd/mail"
)

const modulePath = "github.com/open-cli-collective/google-cli"

var domainPackages = []string{"calendar", "contacts", "drive", "mail", "me"}

func domainCommands() map[string]*cobra.Command {
	return map[string]*cobra.Command{
		"calendar": calcmd.NewCommand(),
		"contacts": contactscmd.NewCommand(),
		"drive":    drivecmd.NewCommand(),
		"mail":     mailcmd.NewCommand(),
		"me":       mecmd.NewCommand(),
	}
}

type commandPair struct {
	read  func() *cobra.Command
	write func() *cobra.Command
}

var writeCommandPairs = map[string]commandPair{
	"mail": {read: mailcmd.NewCommand, write: rwmailcmd.NewCommand},
}

type leafInfo struct {
	path string
	cmd  *cobra.Command
}

func leafCommands(cmd *cobra.Command, parentPath string) []leafInfo {
	if len(cmd.Commands()) == 0 {
		return []leafInfo{{path: parentPath, cmd: cmd}}
	}
	var leaves []leafInfo
	for _, sub := range cmd.Commands() {
		leaves = append(leaves, leafCommands(sub, parentPath+" "+sub.Name())...)
	}
	return leaves
}

func leafMap(cmd *cobra.Command) map[string]*cobra.Command {
	leaves := make(map[string]*cobra.Command)
	for _, leaf := range leafCommands(cmd, cmd.Name()) {
		leaves[leaf.path] = leaf.cmd
	}
	return leaves
}

// assertReadSupersetAndAddedLeaves fails the test if the write command drops
// any read leaf, then returns the leaves only the write command has.
func assertReadSupersetAndAddedLeaves(t *testing.T, domain string) map[string]*cobra.Command {
	t.Helper()
	pair, ok := writeCommandPairs[domain]
	if !ok {
		t.Fatalf("internal/rwcmd/%s needs a command pair in architecture tests", domain)
	}
	read, write := leafMap(pair.read()), leafMap(pair.write())
	for path := range read {
		if _, ok := write[path]; !ok {
			t.Errorf("write command is missing read leaf %q", path)
		}
	}
	for path := range read {
		delete(write, path)
	}
	return write
}

func packageDirs(t *testing.T, kind string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(repoRoot(t), "internal", kind))
	if err != nil {
		t.Fatalf("read internal/%s: %v", kind, err)
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	return names
}

func parseNonTestFiles(t *testing.T, dir string) []*ast.File {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	var files []*ast.File
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		files = append(files, file)
	}
	return files
}

type commandPackage struct {
	name string
	dir  string
}

func commandPackages(t *testing.T) []commandPackage {
	t.Helper()
	root := repoRoot(t)
	packages := make([]commandPackage, 0, len(domainPackages))
	for _, name := range domainPackages {
		packages = append(packages, commandPackage{"cmd/" + name, filepath.Join(root, "internal", "cmd", name)})
	}
	for _, name := range packageDirs(t, "rwcmd") {
		packages = append(packages, commandPackage{"rwcmd/" + name, filepath.Join(root, "internal", "rwcmd", name)})
	}
	return packages
}

func TestDomainPackagesDefineClientInterface(t *testing.T) {
	t.Parallel()
	for _, pkg := range commandPackages(t) {
		pkg := pkg
		t.Run(pkg.name, func(t *testing.T) {
			t.Parallel()
			for _, file := range parseNonTestFiles(t, pkg.dir) {
				for _, decl := range file.Decls {
					gen, ok := decl.(*ast.GenDecl)
					if !ok || gen.Tok != token.TYPE {
						continue
					}
					for _, spec := range gen.Specs {
						typ, ok := spec.(*ast.TypeSpec)
						if _, isInterface := typ.Type.(*ast.InterfaceType); ok && isInterface && typ.Name.IsExported() && strings.HasSuffix(typ.Name.Name, "Client") {
							return
						}
					}
				}
			}
			t.Errorf("internal/%s must define an exported interface ending in Client", pkg.name)
		})
	}
}

func TestDomainPackagesHaveClientFactory(t *testing.T) {
	t.Parallel()
	for _, pkg := range commandPackages(t) {
		pkg := pkg
		t.Run(pkg.name, func(t *testing.T) {
			t.Parallel()
			for _, file := range parseNonTestFiles(t, pkg.dir) {
				for _, decl := range file.Decls {
					gen, ok := decl.(*ast.GenDecl)
					if !ok || gen.Tok != token.VAR {
						continue
					}
					for _, spec := range gen.Specs {
						value, ok := spec.(*ast.ValueSpec)
						if !ok {
							continue
						}
						for _, name := range value.Names {
							if name.Name == "ClientFactory" {
								return
							}
						}
					}
				}
			}
			t.Errorf("internal/%s must define ClientFactory", pkg.name)
		})
	}
}

func TestDomainPackagesExportNewCommand(t *testing.T) {
	t.Parallel()
	for _, pkg := range commandPackages(t) {
		pkg := pkg
		t.Run(pkg.name, func(t *testing.T) {
			t.Parallel()
			for _, file := range parseNonTestFiles(t, pkg.dir) {
				for _, decl := range file.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if ok && fn.Recv == nil && fn.Name.Name == "NewCommand" {
						return
					}
				}
			}
			t.Errorf("internal/%s must export NewCommand", pkg.name)
		})
	}
}

func TestWriteCommandsExtendReadCommands(t *testing.T) {
	t.Parallel()
	for _, domain := range packageDirs(t, "rwcmd") {
		domain := domain
		t.Run(domain, func(t *testing.T) {
			t.Parallel()
			if added := assertReadSupersetAndAddedLeaves(t, domain); len(added) == 0 {
				t.Error("write command must add at least one leaf")
			}
		})
	}
}

// embedsSelector reports whether some exported type accepted by typeName embeds
// a type from importPath. With selectedType set it must be that exact type;
// with selectedType empty any exported type named *Client counts.
func embedsSelector(files []*ast.File, typeName func(string) bool, importPath, selectedType string) bool {
	for _, file := range files {
		aliases := map[string]bool{}
		for _, spec := range file.Imports {
			path, _ := strconv.Unquote(spec.Path.Value)
			if path != importPath {
				continue
			}
			alias := filepath.Base(path)
			if spec.Name != nil {
				alias = spec.Name.Name
			}
			aliases[alias] = true
		}
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typ, ok := spec.(*ast.TypeSpec)
				if !ok || !typ.Name.IsExported() || !typeName(typ.Name.Name) {
					continue
				}
				var fields *ast.FieldList
				switch node := typ.Type.(type) {
				case *ast.StructType:
					fields = node.Fields
				case *ast.InterfaceType:
					fields = node.Methods
				}
				if fields == nil {
					continue
				}
				for _, field := range fields.List {
					if len(field.Names) != 0 {
						continue
					}
					expr := field.Type
					if star, ok := expr.(*ast.StarExpr); ok {
						expr = star.X
					}
					sel, ok := expr.(*ast.SelectorExpr)
					if !ok {
						continue
					}
					ident, identOK := sel.X.(*ast.Ident)
					matchesExactType := selectedType != "" && sel.Sel.Name == selectedType
					matchesClientConvention := selectedType == "" && sel.Sel.IsExported() && strings.HasSuffix(sel.Sel.Name, "Client")
					if identOK && aliases[ident.Name] && (matchesExactType || matchesClientConvention) {
						return true
					}
				}
			}
		}
	}
	return false
}

func TestWriteClientsEmbedReadClients(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, domain := range packageDirs(t, "rw") {
		domain := domain
		t.Run(domain, func(t *testing.T) {
			t.Parallel()
			files := parseNonTestFiles(t, filepath.Join(root, "internal", "rw", domain))
			if !embedsSelector(files, func(name string) bool { return name == "Client" }, modulePath+"/internal/api/"+domain, "Client") {
				t.Errorf("internal/rw/%s.Client must embed *internal/api/%s.Client", domain, domain)
			}
		})
	}
}

func TestWriteCommandClientsEmbedReadCommandClients(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, domain := range packageDirs(t, "rwcmd") {
		domain := domain
		t.Run(domain, func(t *testing.T) {
			t.Parallel()
			files := parseNonTestFiles(t, filepath.Join(root, "internal", "rwcmd", domain))
			if !embedsSelector(files, func(name string) bool { return strings.HasSuffix(name, "Client") }, modulePath+"/internal/cmd/"+domain, "") {
				t.Errorf("internal/rwcmd/%s client interface must embed internal/cmd/%s's client interface", domain, domain)
			}
		})
	}
}

func TestWriteLeavesHaveDryRun(t *testing.T) {
	t.Parallel()
	readVerbs := map[string]bool{"list": true, "get": true, "show": true, "search": true}
	for _, domain := range packageDirs(t, "rwcmd") {
		for path, cmd := range assertReadSupersetAndAddedLeaves(t, domain) {
			path, cmd := path, cmd
			t.Run(path, func(t *testing.T) {
				t.Parallel()
				parts := strings.Fields(path)
				if readVerbs[parts[len(parts)-1]] {
					return
				}
				flag := cmd.Flags().Lookup("dry-run")
				if flag == nil || flag.Value.Type() != "bool" || flag.Shorthand != "n" {
					t.Errorf("write leaf %q must declare bool --dry-run (-n)", path)
				}
			})
		}
	}
}

func TestPermanentWriteLeavesRequireYes(t *testing.T) {
	t.Parallel()
	for _, domain := range packageDirs(t, "rwcmd") {
		for path, cmd := range assertReadSupersetAndAddedLeaves(t, domain) {
			path, cmd := path, cmd
			t.Run(path, func(t *testing.T) {
				t.Parallel()
				if cmd.Flags().Lookup("permanent") != nil && cmd.Flags().Lookup("yes") == nil {
					t.Errorf("write leaf %q has --permanent without --yes", path)
				}
			})
		}
	}
}

func TestResourceLeavesHaveNoJSONFlag(t *testing.T) {
	t.Parallel()
	roots := domainCommands()
	for domain, pair := range writeCommandPairs {
		roots["grw "+domain] = pair.write()
	}
	for name, cmd := range roots {
		for _, leaf := range leafCommands(cmd, name) {
			leaf := leaf
			t.Run(leaf.path, func(t *testing.T) {
				t.Parallel()
				if leaf.cmd.Flags().Lookup("json") != nil {
					t.Errorf("resource leaf %q must not declare --json (see docs/golden-principles.md and cli-common output-and-rendering §2)", leaf.path)
				}
			})
		}
	}
}

func TestResourceLeaf_RejectsJSON_EndToEnd(t *testing.T) {
	t.Parallel()
	cmd := drivecmd.NewCommand()
	cmd.SetArgs([]string{"drives", "--json"})
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("expected unknown flag error, got %v", err)
	}
}

// allowedScopes are the only OAuth scopes gro may request. The non-readonly
// entries are included because they enable organizational operations (label,
// archive, star, RSVP) without granting send or delete access.
var allowedScopes = map[string]bool{
	"https://www.googleapis.com/auth/gmail.readonly":    true,
	"https://www.googleapis.com/auth/gmail.modify":      true, // label, archive, star, read/unread (NOT send/delete)
	"https://www.googleapis.com/auth/calendar.readonly": true,
	"https://www.googleapis.com/auth/calendar.events":   true, // RSVP, color (NOT calendar settings)
	"https://www.googleapis.com/auth/contacts.readonly": true,
	"https://www.googleapis.com/auth/contacts":          true, // group membership, starring (NOT create/delete contacts)
	"https://www.googleapis.com/auth/userinfo.profile":  true, // authenticated user's name/email for `me` (NOT contacts list)
	"https://www.googleapis.com/auth/drive.readonly":    true,
	"https://www.googleapis.com/auth/drive.metadata":    true, // star/unstar files (NOT file content write)
}

var knownGrwScopes = map[string]bool{
	"https://www.googleapis.com/auth/gmail.readonly":       true,
	"https://www.googleapis.com/auth/gmail.modify":         true,
	"https://www.googleapis.com/auth/calendar.readonly":    true,
	"https://www.googleapis.com/auth/calendar.events":      true,
	"https://www.googleapis.com/auth/contacts.readonly":    true,
	"https://www.googleapis.com/auth/contacts":             true,
	"https://www.googleapis.com/auth/userinfo.profile":     true,
	"https://www.googleapis.com/auth/drive.readonly":       true,
	"https://www.googleapis.com/auth/drive.metadata":       true,
	"https://www.googleapis.com/auth/gmail.settings.basic": true,
	"https://mail.google.com/":                             true,
}

// TestAllScopesAreNonDestructive is the structural guarantee that keeps gro
// non-destructive: the scope set it registers can never include
// gmail.settings.* or https://mail.google.com/ (which would permit filters or
// permanent deletion).
func TestAllScopesAreNonDestructive(t *testing.T) {
	t.Parallel()
	scopes := groapp.Identity().Scopes
	if len(scopes) == 0 {
		t.Fatal("gro scopes must not be empty")
	}
	for _, scope := range scopes {
		if !allowedScopes[scope] {
			t.Errorf("scope %q is not in the non-destructive allowlist; update allowedScopes if this scope is safe", scope)
		}
	}
}

func scopeService(scope string) string {
	const marker = "/auth/"
	_, suffix, ok := strings.Cut(scope, marker)
	if !ok {
		return ""
	}
	service, _, _ := strings.Cut(suffix, ".")
	return service
}

func TestGrwScopesCoverGroPerService(t *testing.T) {
	t.Parallel()
	grwScopes := grwapp.Identity().Scopes
	grwSet, services := map[string]bool{}, map[string]bool{}
	for _, scope := range grwScopes {
		grwSet[scope] = true
		if service := scopeService(scope); service != "" {
			services[service] = true
		}
	}
	for _, scope := range groapp.Identity().Scopes {
		if services[scopeService(scope)] && !grwSet[scope] {
			t.Errorf("grw scopes include service %q but omit gro scope %q", scopeService(scope), scope)
		}
	}
}

func TestGrwScopesAreKnown(t *testing.T) {
	t.Parallel()
	scopes := grwapp.Identity().Scopes
	if len(scopes) == 0 {
		t.Fatal("grw scopes must not be empty")
	}
	for _, scope := range scopes {
		if !knownGrwScopes[scope] {
			t.Errorf("grw scope %q is not in the known allowlist", scope)
		}
	}
}

type listedPackage struct {
	ImportPath string
	Dir        string
	GoFiles    []string
	Module     *struct{ Path string }
}

func goListDeps(t *testing.T, target string) []listedPackage {
	t.Helper()
	cmd := exec.Command("go", "list", "-deps", "-json", target)
	cmd.Dir = repoRoot(t)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list %s: %v\n%s", target, err, stderr.String())
	}
	decoder := json.NewDecoder(bytes.NewReader(out))
	var packages []listedPackage
	for {
		var pkg listedPackage
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("decode go list %s: %v", target, err)
		}
		packages = append(packages, pkg)
	}
	return packages
}

func TestGroNeverLinksWriteCode(t *testing.T) {
	t.Parallel()
	for _, pkg := range goListDeps(t, "./cmd/gro") {
		if strings.HasPrefix(pkg.ImportPath, modulePath+"/internal/rw/") || strings.HasPrefix(pkg.ImportPath, modulePath+"/internal/rwcmd/") {
			t.Errorf("gro links write package %s", pkg.ImportPath)
		}
	}
}

func TestGrwLinksWriteCode(t *testing.T) {
	t.Parallel()
	want := map[string]bool{
		modulePath + "/internal/rw/gmail":   false,
		modulePath + "/internal/rwcmd/mail": false,
	}
	for _, pkg := range goListDeps(t, "./cmd/grw") {
		if _, ok := want[pkg.ImportPath]; ok {
			want[pkg.ImportPath] = true
		}
	}
	for pkg, found := range want {
		if !found {
			t.Errorf("grw does not link required write package %s", pkg)
		}
	}
}

// forbiddenAPIPatterns are the Google API client method calls gro must never
// make. They are specific to the Google client libraries and unlikely to
// appear in other contexts; generic names like .Delete() or .Insert() are
// intentionally excluded to avoid false positives. .BatchModify( is allowed
// because it backs bulk label/archive operations.
var forbiddenAPIPatterns = []string{".Send(", ".Trash(", ".Untrash(", ".BatchDelete("}

// TestNoDestructiveAPIMethodsInProductionCode scans every in-module package
// that gro links (including internal/api/*) for the forbidden calls. Scoping
// the scan to gro's link graph is deliberate: internal/rw/* legitimately calls
// these methods, and TestGroNeverLinksWriteCode proves gro cannot reach it.
func TestNoDestructiveAPIMethodsInProductionCode(t *testing.T) {
	t.Parallel()
	root := repoRoot(t)
	for _, pkg := range goListDeps(t, "./cmd/gro") {
		if pkg.Module == nil || pkg.Module.Path != modulePath {
			continue
		}
		for _, name := range pkg.GoFiles {
			path := filepath.Join(pkg.Dir, name)
			data, err := os.ReadFile(path) //nolint:gosec // package paths come from go list
			if err != nil {
				t.Errorf("read %s: %v", path, err)
				continue
			}
			for _, pattern := range forbiddenAPIPatterns {
				if strings.Contains(string(data), pattern) {
					rel, _ := filepath.Rel(root, path)
					t.Errorf("%s contains forbidden destructive API method %q; gro only allows non-destructive operations", rel, pattern)
				}
			}
		}
	}
}
