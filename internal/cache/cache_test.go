package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/open-cli-collective/cli-common/statedirtest"

	"github.com/open-cli-collective/google-cli/internal/config"
	"github.com/open-cli-collective/google-cli/internal/keychain"
	"github.com/open-cli-collective/google-cli/internal/testutil"
)

// hermetic isolates the §3.1 7-var env set so os.UserCacheDir /
// os.UserConfigDir resolve under a per-test temp root on every OS.
func hermetic(t *testing.T) {
	t.Helper()
	statedirtest.Hermetic(t)
}

func TestNew(t *testing.T) {
	hermetic(t)
	t.Run("creates cache", func(t *testing.T) {
		c, err := New()
		testutil.NoError(t, err)
		testutil.NotNil(t, c)
		defer c.Clear()
	})

	t.Run("creates cache directory", func(t *testing.T) {
		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()

		_, err = os.Stat(c.loc.Root)
		testutil.NoError(t, err)
	})
}

func TestCache_GetSetDrives(t *testing.T) {
	hermetic(t)
	c, err := New()
	testutil.NoError(t, err)
	defer c.Clear()

	t.Run("returns nil for missing cache", func(t *testing.T) {
		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Nil(t, drives)
	})

	t.Run("stores and retrieves drives", func(t *testing.T) {
		input := []*CachedDrive{
			{ID: "drive1", Name: "Engineering"},
			{ID: "drive2", Name: "Marketing"},
		}

		err := c.SetDrives(input)
		testutil.NoError(t, err)

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Len(t, drives, 2)
		testutil.Equal(t, drives[0].ID, "drive1")
		testutil.Equal(t, drives[0].Name, "Engineering")
		testutil.Equal(t, drives[1].ID, "drive2")
		testutil.Equal(t, drives[1].Name, "Marketing")
	})
}

func TestCacheIsolatedBySelectedProfile(t *testing.T) {
	hermetic(t)
	keychain.SetCredentialRefOverride("google-readonly/default", true)
	t.Cleanup(func() { keychain.SetCredentialRefOverride("", false) })
	defaultCache, err := New()
	testutil.NoError(t, err)
	testutil.NoError(t, defaultCache.SetDrives([]*CachedDrive{{ID: "default", Name: "Default"}}))

	keychain.SetCredentialRefOverride("google-readonly/work", true)
	workCache, err := New()
	testutil.NoError(t, err)
	got, err := workCache.GetDrives()
	testutil.NoError(t, err)
	testutil.Nil(t, got)
	testutil.NoError(t, workCache.SetDrives([]*CachedDrive{{ID: "work", Name: "Work"}}))

	keychain.SetCredentialRefOverride("google-readonly/default", true)
	got, err = defaultCache.GetDrives()
	testutil.NoError(t, err)
	testutil.Len(t, got, 1)
	testutil.Equal(t, got[0].ID, "default")
	defaultCache.Clear()
	workCache.Clear()
}

func TestCache_Expiration(t *testing.T) {
	hermetic(t)
	c, err := New()
	testutil.NoError(t, err)
	defer c.Clear()

	t.Run("classifies stale envelope as miss", func(t *testing.T) {
		testutil.NoError(t, c.SetDrives([]*CachedDrive{{ID: "x", Name: "y"}}))

		// Pretend "now" is two days ahead of the envelope's FetchedAt (TTL is 24h).
		origNow := nowFn
		nowFn = func() time.Time { return time.Now().Add(48 * time.Hour) }
		defer func() { nowFn = origNow }()

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Nil(t, drives)
	})

	t.Run("returns drives for fresh envelope", func(t *testing.T) {
		testutil.NoError(t, c.SetDrives([]*CachedDrive{{ID: "drive1", Name: "Test"}}))
		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Len(t, drives, 1)
		testutil.Equal(t, drives[0].ID, "drive1")
	})
}

func TestCache_CorruptedCache(t *testing.T) {
	hermetic(t)
	c, err := New()
	testutil.NoError(t, err)
	defer c.Clear()

	t.Run("malformed JSON treated as miss (disposable state)", func(t *testing.T) {
		// Write a malformed envelope at the same path cli-common would use:
		// <Root>/<InstanceKey>/<resource>.json.
		path := filepath.Join(c.loc.Root, c.loc.InstanceKey, drivesResource+".json")
		testutil.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		testutil.NoError(t, os.WriteFile(path, []byte("not valid json"), 0o600))

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Nil(t, drives)
	})

	t.Run("old pre-MON-5371 DriveCache shape treated as miss", func(t *testing.T) {
		// cli-common envelope expects Version=1, Resource="drives",
		// Instance="default"; the pre-MON-5371 shape supplied none of those,
		// so ReadResource classifies it as ErrCacheMiss.
		path := filepath.Join(c.loc.Root, c.loc.InstanceKey, drivesResource+".json")
		testutil.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		oldShape := `{"cached_at":"2026-01-01T00:00:00Z","ttl_hours":24,"drives":[{"id":"d1","name":"x"}]}`
		testutil.NoError(t, os.WriteFile(path, []byte(oldShape), 0o600))

		drives, err := c.GetDrives()
		testutil.NoError(t, err)
		testutil.Nil(t, drives)
	})
}

func TestCache_DrivesStatus(t *testing.T) {
	hermetic(t)

	t.Run("uninitialized when no envelope on disk", func(t *testing.T) {
		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()

		fetchedAt, ttl, status, now, err := c.DrivesStatus()
		testutil.NoError(t, err)
		testutil.True(t, fetchedAt.IsZero())
		testutil.True(t, !now.IsZero())
		testutil.Equal(t, ttl, drivesTTL)
		testutil.Equal(t, status.String(), "uninitialized")
	})

	t.Run("uninitialized when envelope is malformed JSON (corrupt-as-miss)", func(t *testing.T) {
		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()

		path := filepath.Join(c.loc.Root, c.loc.InstanceKey, drivesResource+".json")
		testutil.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
		testutil.NoError(t, os.WriteFile(path, []byte("not valid json"), 0o600))

		_, _, status, _, err := c.DrivesStatus()
		testutil.NoError(t, err)
		testutil.Equal(t, status.String(), "uninitialized")
	})

	t.Run("propagates generic I/O errors (envelope not a regular file)", func(t *testing.T) {
		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()

		// Make the envelope path a directory; cli-common/cache's reader will
		// fail with a non-syntax I/O error that must propagate from DrivesStatus.
		path := filepath.Join(c.loc.Root, c.loc.InstanceKey, drivesResource+".json")
		testutil.NoError(t, os.MkdirAll(path, 0o700))

		_, _, _, _, err = c.DrivesStatus()
		if err == nil {
			t.Fatal("expected I/O error to propagate, got nil")
		}
	})

	t.Run("fresh after SetDrives", func(t *testing.T) {
		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()

		testutil.NoError(t, c.SetDrives([]*CachedDrive{{ID: "d1", Name: "X"}}))

		fetchedAt, ttl, status, _, err := c.DrivesStatus()
		testutil.NoError(t, err)
		testutil.True(t, !fetchedAt.IsZero())
		testutil.Equal(t, ttl, drivesTTL)
		testutil.Equal(t, status.String(), "fresh")
	})

	t.Run("stale when fetchedAt is older than TTL", func(t *testing.T) {
		c, err := New()
		testutil.NoError(t, err)
		defer c.Clear()

		testutil.NoError(t, c.SetDrives([]*CachedDrive{{ID: "d1", Name: "X"}}))

		// Advance the cache clock past the 24h drives TTL.
		origNow := nowFn
		nowFn = func() time.Time { return time.Now().Add(48 * time.Hour) }
		defer func() { nowFn = origNow }()

		_, _, status, _, err := c.DrivesStatus()
		testutil.NoError(t, err)
		testutil.Equal(t, status.String(), "stale")
	})
}

func TestCache_Clear(t *testing.T) {
	hermetic(t)
	c, err := New()
	testutil.NoError(t, err)

	testutil.NoError(t, c.SetDrives([]*CachedDrive{{ID: "test", Name: "Test"}}))

	testutil.NoError(t, c.Clear())
	// Clear scoped to <Root>/<InstanceKey>, not the tool-level Root.
	_, err = os.Stat(filepath.Join(c.loc.Root, c.loc.InstanceKey))
	testutil.True(t, os.IsNotExist(err))
}

func TestCache_GetDir(t *testing.T) {
	hermetic(t)
	c, err := New()
	testutil.NoError(t, err)
	defer c.Clear()

	want, err := config.CacheDirPath()
	testutil.NoError(t, err)
	testutil.NotEmpty(t, c.GetDir())
	testutil.Equal(t, c.GetDir(), want)
}

func TestLegacyCacheIsDisposable(t *testing.T) {
	hermetic(t)
	legacy, err := config.LegacyCacheDir()
	testutil.NoError(t, err)
	testutil.NoError(t, os.MkdirAll(legacy, 0o700))
	testutil.NoError(t, os.WriteFile(filepath.Join(legacy, DrivesFile),
		[]byte(`{"drives":[{"id":"old","name":"Stale"}]}`), 0o600))

	c, err := New()
	testutil.NoError(t, err)
	defer c.Clear()
	got, err := c.GetDrives()
	testutil.NoError(t, err)
	testutil.Nil(t, got)
	// The old unscoped cache is neither guessed into this profile nor removed
	// by normal cache construction; config clear --all owns broad cleanup.
	_, statErr := os.Stat(filepath.Join(legacy, DrivesFile))
	testutil.NoError(t, statErr)
}
