// Package profiles implements the profiles command group: the multi-
// account visibility surface. Its reason to exist is an incident: with one
// stale token on the active profile, every command failed with an oauth2
// error, and discovering that OTHER (healthy) profiles existed required
// dumping the OS keychain by hand. `profiles list` makes stored profiles,
// their accounts, and their token state first-class; `profiles use` makes
// switching the active binding deliberate and visible instead of a
// hand-edit of config.yml.
package profiles

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/google-cli/internal/api/gmail"
	"github.com/open-cli-collective/google-cli/internal/auth"
	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/identitycache"
	"github.com/open-cli-collective/google-cli/internal/keychain"
	"github.com/open-cli-collective/google-cli/internal/output"
	"github.com/open-cli-collective/google-cli/internal/sanitize"
)

// NewCommand returns the profiles command with subcommands.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "List and switch credential profiles",
		Long: `Manage the credential profiles stored in the OS keyring.

A profile holds one Google account's OAuth token. The active profile is the
credential_ref in config.yml (overridable per invocation with --profile or
--ref, or with the <SERVICE>_CREDENTIAL_REF environment variable).`,
	}
	cmd.AddCommand(newListCommand())
	cmd.AddCommand(newUseCommand())
	cmd.AddCommand(newRenameCommand())
	return cmd
}

// OpenStore is the keychain seam (var so tests can substitute). OpenNoMigrate:
// profiles list is a diagnostic surface and must stay usable during an
// unresolved §1.8 migration conflict — exactly when the user most needs to
// see what exists.
var OpenStore = keychain.OpenNoMigrate

// OpenRefStore opens the source profile explicitly. Rename must use this
// seam rather than the active store so a global selector cannot redirect the
// positional source.
var OpenRefStore = keychain.OpenRef

var (
	// These seams keep the failure ordering testable without touching a real
	// keyring or config file. The production path still uses the concrete
	// credstore-backed operations directly.
	renameCopy = func(st *keychain.Store, oldProfile, newProfile string) error {
		return st.CopyProfile(oldProfile, newProfile)
	}
	renameDelete     = func(st *keychain.Store, profile string) error { return st.DeleteProfile(profile) }
	renameSaveConfig = config.SaveConfig
	renameIdentity   = identitycache.Rename
)

// VerifyRef live-verifies one profile's token by asking the Gmail profile
// for its email (gmail scope is granted by every CLI built on this library).
// Var so tests can substitute.
var VerifyRef = func(ctx context.Context, ref string) (string, error) {
	c, err := gmail.NewClientForRef(ctx, ref)
	if err != nil {
		return "", err
	}
	p, err := c.GetProfile(ctx)
	if err != nil {
		return "", err
	}
	return p.EmailAddress, nil
}

// profileRow is one profile's non-secret listing state (§1.6: presence and
// metadata only, never token material).
type profileRow struct {
	Profile      string `json:"profile"`
	Ref          string `json:"ref"`
	Active       bool   `json:"active"`
	TokenPresent bool   `json:"token_present"`
	Email        string `json:"email,omitempty"`
	VerifiedAt   string `json:"verified_at,omitempty"`
	Health       string `json:"health,omitempty"` // only with --check
}

func newListCommand() *cobra.Command {
	var jsonOut, check bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List credential profiles and their token state",
		Long: `List every profile stored in the OS keyring for this CLI, the account
email each holds (as last verified), and whether a token is present. The
active profile is marked with '*'.

--check live-verifies each stored token against the API and reports
  ok / expired / error per profile (one API round-trip per profile), updating
  the cached email as a side effect. Use it to answer "which of my accounts
  still work?" without trying them one by one.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runList(cmd.Context(), jsonOut, check)
		},
	}
	cmd.Flags().BoolVarP(&jsonOut, "json", "j", false, "Emit JSON")
	cmd.Flags().BoolVar(&check, "check", false, "Live-verify each profile's token against the API")
	return cmd
}

func runList(ctx context.Context, jsonOut, check bool) error {
	st, err := OpenStore()
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	profiles, err := st.ListProfiles()
	if err != nil {
		return err
	}

	activeRef := st.Ref()
	service := st.Service()

	// The active profile is listed even before it has any stored keys (fresh
	// install, or `profiles use` pointing at a not-yet-authenticated name):
	// hiding the pointer's target would be this command failing its own
	// purpose.
	_, activeProfile, perr := credstore.ParseRef(activeRef)
	if perr == nil && !contains(profiles, activeProfile) {
		profiles = append(profiles, activeProfile)
		sort.Strings(profiles)
	}

	cached := identitycache.Load()
	rows := make([]profileRow, 0, len(profiles))
	for _, p := range profiles {
		ref := service + "/" + p
		present, herr := st.HasTokenFor(p)
		if herr != nil {
			return herr
		}
		row := profileRow{
			Profile:      p,
			Ref:          ref,
			Active:       ref == activeRef,
			TokenPresent: present,
		}
		if e, ok := cached[p]; ok {
			row.Email = e.Email
			row.VerifiedAt = e.VerifiedAt.Format(time.RFC3339)
		}
		if check {
			health, email, verifiedAt := checkProfile(ctx, p, ref, present)
			row.Health = health
			if email != "" {
				row.Email = email
				row.VerifiedAt = verifiedAt.Format(time.RFC3339)
			}
		}
		rows = append(rows, row)
	}

	if jsonOut {
		return output.JSONStdout(rows)
	}

	w := tabwriter.NewWriter(os.Stdout, 2, 4, 2, ' ', 0)
	header := "ACTIVE\tPROFILE\tTOKEN\tEMAIL"
	if check {
		header += "\tHEALTH"
	}
	_, _ = fmt.Fprintln(w, header)
	for _, r := range rows {
		marker := ""
		if r.Active {
			marker = "*"
		}
		email := r.Email
		if email == "" {
			email = "-"
		}
		line := fmt.Sprintf("%s\t%s\t%s\t%s", marker, r.Profile, presence(r.TokenPresent), sanitize.Output(email))
		if check {
			line += "\t" + r.Health
		}
		_, _ = fmt.Fprintln(w, line)
	}
	_ = w.Flush()

	prod := config.ProductName()
	fmt.Println()
	fmt.Printf("Active: %s (via %s)\n", activeRef, keychain.DescribeRefSource(st.RefSource()))
	fmt.Printf("Switch with '%s profiles use <profile>', or per invocation with --profile <name>.\n", prod)
	for _, r := range rows {
		if r.Active && !r.TokenPresent {
			fmt.Printf("The active profile has no stored token - run '%s init' to authenticate it.\n", prod)
			break
		}
	}
	return nil
}

// checkProfile classifies one profile's live token state, returning the
// health label plus the verified email/time (both zero unless healthy). The
// email learned from a healthy check also refreshes the identity cache
// (best-effort), so --check is how a listing heals missing/stale emails.
func checkProfile(ctx context.Context, profile, ref string, present bool) (health, email string, verifiedAt time.Time) {
	if !present {
		return "no token", "", time.Time{}
	}
	email, err := VerifyRef(ctx, ref)
	if err != nil {
		if auth.IsAuthError(err) {
			return "expired or revoked", "", time.Time{}
		}
		return "error: " + firstLine(err.Error()), "", time.Time{}
	}
	if cerr := identitycache.Put(profile, email); cerr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not cache identity for %s: %v\n", profile, cerr)
	}
	return "ok", email, time.Now().UTC()
}

func newUseCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use <profile>",
		Short: "Switch the active credential profile",
		Long: `Set config.yml's credential_ref to the given profile, making it the
account every subsequent command uses. Accepts a bare profile name ("work")
or a full <service>/<profile> ref for this CLI.

Switching never touches tokens: the previous profile's token stays stored
and can be switched back to at any time. A per-invocation --ref flag or the
<SERVICE>_CREDENTIAL_REF environment variable still takes precedence over
the switched binding.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runUse(args[0])
		},
	}
	return cmd
}

func runUse(arg string) error {
	st, err := OpenStore()
	if err != nil {
		return err
	}
	service := st.Service()

	profile := arg
	if strings.Contains(arg, "/") {
		svc, p, perr := credstore.ParseRef(arg)
		if perr != nil {
			_ = st.Close()
			return fmt.Errorf("invalid ref %q: %w", arg, perr)
		}
		if svc != service {
			_ = st.Close()
			return fmt.Errorf("ref %q belongs to service %q, not this CLI's %q — cross-service refs would point %s at another tool's credentials",
				arg, svc, service, config.ProductName())
		}
		profile = p
	}
	ref, err := credstore.FormatRef(service, profile)
	if err != nil {
		_ = st.Close()
		return fmt.Errorf("invalid profile %q: %w", profile, err)
	}

	known, err := st.ListProfiles()
	if err != nil {
		_ = st.Close()
		return err
	}
	hasToken, err := st.HasTokenFor(profile)
	if err != nil {
		_ = st.Close()
		return err
	}
	_ = st.Close()

	prod := config.ProductName()
	// An unknown/token-less profile is allowed — `use work` then `init` is
	// the natural add-a-second-account flow — but loudly, so a typo doesn't
	// silently bind commands to an empty profile.
	if !hasToken {
		fmt.Printf("Note: no token is stored for %s yet.\n", ref)
		if len(known) > 0 {
			fmt.Printf("Existing profiles: %s\n", strings.Join(known, ", "))
		}
		fmt.Printf("Run '%s init' after switching to authenticate it.\n", prod)
	}

	cfg, err := config.LoadConfigForRuntime()
	if err != nil {
		return err
	}
	if cfg.CredentialRef == ref {
		fmt.Printf("%s is already the active profile.\n", ref)
		return nil
	}
	cfg.CredentialRef = ref
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	fmt.Printf("Active profile is now %s.\n", ref)

	if env := os.Getenv(keychain.CredentialRefEnvVar()); env != "" && env != ref {
		fmt.Printf("Note: %s=%s is set in this shell and overrides the switched profile.\n",
			keychain.CredentialRefEnvVar(), env)
	}
	return nil
}

func newRenameCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a credential profile",
		Long: `Rename a profile in this CLI's credential namespace without
re-authenticating it. The destination must not already contain credentials;
other profiles remain unchanged. The saved active profile is updated when it
points at the old name.`,
		Args: cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			return runRename(args[0], args[1])
		},
	}
	return cmd
}

func runRename(oldProfile, newProfile string) error {
	service, _, err := credstore.ParseRef(config.DefaultCredentialRef)
	if err != nil {
		return fmt.Errorf("resolve CLI service: %w", err)
	}
	oldRef, err := credstore.FormatRef(service, oldProfile)
	if err != nil {
		return fmt.Errorf("invalid old profile %q: %w", oldProfile, err)
	}
	newRef, err := credstore.FormatRef(service, newProfile)
	if err != nil {
		return fmt.Errorf("invalid new profile %q: %w", newProfile, err)
	}

	// Load the persisted binding before touching credentials. Runtime selector
	// overrides are intentionally absent here: rename's positional old name is
	// always the source, and only the saved config binding may be rewritten.
	cfg, err := config.LoadConfigForRuntime()
	if err != nil {
		return err
	}
	st, err := OpenRefStore(oldRef)
	if err != nil {
		return err
	}
	defer func() { _ = st.Close() }()

	if oldProfile == newProfile {
		if err := renameCopy(st, oldProfile, newProfile); err != nil {
			return err
		}
		fmt.Printf("Profile %s is already named %s.\n", oldRef, newRef)
		return nil
	}

	// Copy first: SetBundle validates every key and rolls back partial writes;
	// the source is retained when the copy or any later state update fails.
	if err := renameCopy(st, oldProfile, newProfile); err != nil {
		return err
	}

	activeChanged := cfg.CredentialRef == oldRef
	if activeChanged {
		cfg.CredentialRef = newRef
		cfg.SetCredentialRefSource(config.RefSourceConfig)
		if err := renameSaveConfig(cfg); err != nil {
			// The source is still intact, so remove the copy before returning.
			// That makes a transient config failure retryable while preserving
			// the token if rollback itself cannot complete.
			if rollbackErr := renameDelete(st, newProfile); rollbackErr != nil {
				return fmt.Errorf("saving active profile %s failed after copying credentials; source was retained and copied destination may remain: %w (rollback failed: %w)", oldRef, err, rollbackErr)
			}
			return fmt.Errorf("saving active profile %s failed after copying credentials; source was retained and copied destination was removed: %w", oldRef, err)
		}
	}

	// Delete only after the destination and any required config update are in
	// place. A partial delete leaves the destination copy, so no token is lost.
	if err := renameDelete(st, oldProfile); err != nil {
		return fmt.Errorf("profile copied to %s but source %s could not be removed: %w", newRef, oldRef, err)
	}

	// Identity data is disposable, but preserving its verification timestamp
	// makes the rename transparent to `profiles list`.
	if err := renameIdentity(oldProfile, newProfile); err != nil {
		fmt.Fprintf(os.Stderr, "warning: credentials renamed from %s to %s but cached identity was not moved: %v\n", oldRef, newRef, err)
	}

	fmt.Printf("Renamed profile %s to %s.\n", oldRef, newRef)
	if activeChanged {
		fmt.Printf("Active profile is now %s.\n", newRef)
	}
	if env := os.Getenv(keychain.CredentialRefEnvVar()); env == oldRef {
		fmt.Printf("Note: %s still points to %s; update it in this shell.\n", keychain.CredentialRefEnvVar(), oldRef)
	}
	return nil
}

func presence(ok bool) string {
	if ok {
		return "present"
	}
	return "missing"
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// firstLine truncates a (possibly multi-line, possibly huge) error to its
// first line so the HEALTH column stays a column.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	const maxLen = 80
	if len(s) > maxLen {
		s = s[:maxLen] + "…"
	}
	return s
}
