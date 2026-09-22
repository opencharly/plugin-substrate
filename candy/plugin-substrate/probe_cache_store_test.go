package substratekind

import (
	"testing"
	"time"

	"github.com/opencharly/spec/cache"
	"github.com/opencharly/spec/spec"
)

// probe_cache_store_test.go — exercises the MIGRATED guest-probe cache path
// (probeCacheStore → cache.OpenNamed("probes") + Store.ReadTTL/WriteValue).
// It fails without the migration: the old top-level cache.Read/Write API no
// longer exists, and the named-store layout is asserted here.

// TestProbeCacheStoreNamedAndRoundTrip pins the store identity and the
// read/write round-trip through the shared Store.
func TestProbeCacheStoreNamedAndRoundTrip(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CHARLY_CACHE_DIR", root)

	store := probeCacheStore()
	if want := root + "/probes"; store.Dir() != want {
		t.Fatalf("probeCacheStore().Dir() = %q, want %q", store.Dir(), want)
	}
	// The changed path executes live: write through the Store, read it back.
	results := []spec.ToolStatus{{Name: "gh", Status: "available"}}
	store.WriteValue("container|script", results)
	var got []spec.ToolStatus
	if !store.ReadTTL("container|script", probeCacheTTL, &got) {
		t.Fatal("ReadTTL: a just-written entry must be a hit")
	}
	if len(got) != 1 || got[0].Name != "gh" || got[0].Status != "available" {
		t.Fatalf("round-trip = %+v", got)
	}
	// A different key misses.
	if store.ReadTTL("other", probeCacheTTL, &got) {
		t.Fatal("a different key must miss")
	}
}

// TestProbeCacheStoreTTLExpiry pins the 30s TTL via the Store's backdating seam.
func TestProbeCacheStoreTTLExpiry(t *testing.T) {
	t.Setenv("CHARLY_CACHE_DIR", t.TempDir())
	store := probeCacheStore()
	store.WriteValue("k", []spec.ToolStatus{{Name: "x"}})
	e, ok := store.Get("k")
	if !ok {
		t.Fatal("entry missing after write")
	}
	e.Resolved = time.Now().Add(-2 * probeCacheTTL)
	store.PutEntry("k", e)
	var got []spec.ToolStatus
	if store.ReadTTL("k", probeCacheTTL, &got) {
		t.Fatal("an entry past the TTL must miss")
	}
	_ = cache.OpenNamed // keep the import explicit (the migrated idiom)
}
