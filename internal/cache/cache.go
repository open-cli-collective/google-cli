// Package cache wraps cli-common/cache for gro's Drive metadata cache.
//
// Per cli-common/docs/working-with-state.md §4, gro's cache is disposable
// state at os.UserCacheDir()/google-readonly (via statedir.Cache). Writes are
// atomic via cli-common/cache's temp+rename envelope; TTL is hard-coded per
// resource (no user-configurable cache_ttl_hours — §4.4); reads classify a
// version/identity mismatch as a miss so schema bumps self-heal. The pre-B2b
// "<configdir>/cache/" relocation is retained for installs that pre-date the
// B2b cache move.
package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	clicache "github.com/open-cli-collective/cli-common/cache"
	"github.com/open-cli-collective/cli-common/credstore"

	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/keychain"
)

const (
	// DrivesFile is the cache file for shared drives.
	DrivesFile = "drives.json"
	// drivesResource is the cli-common cache resource name. The on-disk file
	// becomes <cachedir>/<instanceKey>/drives.json.
	drivesResource = "drives"
	// drivesTTL is the §4.4 hard-coded per-resource TTL for shared drives —
	// same 24-hour default the user-configurable knob previously defaulted to.
	drivesTTL = "24h"
)

// CachedDrive represents a cached shared drive entry. Public so callers
// (drives.go) can populate it directly.
type CachedDrive struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Cache is gro's wrapper around the cli-common envelope cache.
type Cache struct {
	loc clicache.Locator
}

// New creates a new Cache instance rooted at the OS cache dir (B2b via
// cli-common/statedir), isolated by the selected bare credential profile.
// Legacy unscoped cache is intentionally ignored: it is disposable and cannot
// be safely attributed to any profile.
func New() (*Cache, error) {
	ref, err := keychain.ResolveEffectiveCredentialRef()
	if err != nil {
		// Package-level tests that construct infrastructure before registering an
		// identity have no effective ref; keep their cache hermetic while real
		// binaries always register one before command execution.
		if config.DefaultCredentialRef == "" {
			ref = "test/default"
		} else {
			return nil, err
		}
	}
	_, profile, err := credstore.ParseRef(ref)
	if err != nil {
		return nil, fmt.Errorf("invalid credential ref %q: %w", ref, err)
	}
	cacheDir, err := config.GetCacheDir()
	if err != nil {
		return nil, err
	}
	return &Cache{
		loc: clicache.Locator{Root: cacheDir, InstanceKey: profile},
	}, nil
}

// GetDrives returns cached shared drives, or nil if cache is stale, missing,
// or corrupt. Corrupt-as-miss preserves the pre-MON-5371 behavior: caches
// are disposable, so a JSON parse error self-heals on the next API call. I/O
// errors (read failure, permission denied) propagate.
func (c *Cache) GetDrives() ([]*CachedDrive, error) {
	env, err := clicache.ReadResource[[]*CachedDrive](c.loc, drivesResource)
	switch {
	case errors.Is(err, clicache.ErrCacheMiss):
		return nil, nil
	case err != nil:
		var syn *json.SyntaxError
		var ute *json.UnmarshalTypeError
		if errors.As(err, &syn) || errors.As(err, &ute) {
			return nil, nil // corrupt → miss (self-heals on next write)
		}
		return nil, fmt.Errorf("reading drives cache: %w", err)
	}

	if clicache.Classify(env.FetchedAt, env.TTL, nowFn()) == clicache.StatusStale {
		return nil, nil // stale → miss
	}
	return env.Data, nil
}

// SetDrives atomically writes the drives cache with the §4.4 hard-coded TTL.
func (c *Cache) SetDrives(drives []*CachedDrive) error {
	if err := clicache.WriteResource(c.loc, drivesResource, drivesTTL, drives); err != nil {
		return fmt.Errorf("writing drives cache: %w", err)
	}
	return nil
}

// DrivesStatus reports the freshness of the cached drives entry without
// fetching from the API. Returns (fetchedAt, ttl, status, now). A missing
// or corrupt envelope returns (time.Time{}, drivesTTL, StatusUninitialized,
// now); I/O errors propagate (in which case `status` is meaningless — callers
// MUST check err first). The TTL string is the hard-coded §4.4 value so the
// `refresh --status` table can render it without callers re-deriving it.
// `now` is the clock used for classification so callers can derive an Age
// column from the same instant — this matters in tests that swap nowFn.
func (c *Cache) DrivesStatus() (time.Time, string, clicache.Status, time.Time, error) {
	now := nowFn()
	env, err := clicache.ReadResource[[]*CachedDrive](c.loc, drivesResource)
	switch {
	case errors.Is(err, clicache.ErrCacheMiss):
		return time.Time{}, drivesTTL, clicache.StatusUninitialized, now, nil
	case err != nil:
		var syn *json.SyntaxError
		var ute *json.UnmarshalTypeError
		if errors.As(err, &syn) || errors.As(err, &ute) {
			return time.Time{}, drivesTTL, clicache.StatusUninitialized, now, nil
		}
		return time.Time{}, drivesTTL, clicache.StatusUninitialized, now, fmt.Errorf("reading drives cache: %w", err)
	}
	return env.FetchedAt, drivesTTL, clicache.Classify(env.FetchedAt, env.TTL, now), now, nil
}

// Clear removes all cached data for this instance. Scoped to
// <Root>/<InstanceKey> rather than the tool-level Root so a future move to a
// multi-instance Locator can't have one instance's Clear() silently evict
// every other instance's cache.
func (c *Cache) Clear() error {
	return os.RemoveAll(filepath.Join(c.loc.Root, c.loc.InstanceKey))
}

// GetDir returns the cache directory path.
func (c *Cache) GetDir() string {
	return c.loc.Root
}

// nowFn is the clock used for cache freshness classification; mutated in
// in-package tests only.
var nowFn = func() time.Time { return time.Now() }
