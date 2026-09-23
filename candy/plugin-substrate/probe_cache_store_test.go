package substratekind

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/opencharly/spec/cache"
	"github.com/opencharly/spec/spec"
)

// probe_cache_store_test.go — exercises the guest-probe cache path on the
// ArtifactStore (probeCacheStore → cache.OpenNamedLayout("probes") +
// Layout.Get/Put with Entry{Payload}). It fails without the migration: the
// probe path must read a fresh entry, miss a stale one, and round-trip the
// TTL through the OCI-layout store.

// TestProbeCacheStoreNamedAndRoundTrip pins the store identity and the
// read/write round-trip through the shared ArtifactStore.
func TestProbeCacheStoreNamedAndRoundTrip(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CHARLY_CACHE_DIR", root)

	store := probeCacheStore()
	if want := root + "/probes"; store.Dir() != want {
		t.Fatalf("probeCacheStore().Dir() = %q, want %q", store.Dir(), want)
	}
	// The changed path executes live: write through the store, read it back.
	results := []spec.ToolStatus{{Name: "gh", Status: "available"}}
	raw, _ := json.Marshal(results)
	if err := store.Put("container|script", cache.Entry{Payload: raw}); err != nil {
		t.Fatal(err)
	}
	e, ok := store.Get("container|script")
	if !ok || !e.FreshTTL(probeCacheTTL) {
		t.Fatal("a just-written entry must be a fresh hit")
	}
	var got []spec.ToolStatus
	if !e.Decode(&got) {
		t.Fatal("decode round-trip failed")
	}
	if len(got) != 1 || got[0].Name != "gh" || got[0].Status != "available" {
		t.Fatalf("round-trip = %+v", got)
	}
	// A different key misses.
	if _, ok := store.Get("other"); ok {
		t.Fatal("a different key must miss")
	}
}

// TestProbeCacheStoreTTLExpiry pins the 30s TTL through the ArtifactStore's
// resolved-time freshness.
func TestProbeCacheStoreTTLExpiry(t *testing.T) {
	t.Setenv("CHARLY_CACHE_DIR", t.TempDir())
	store := probeCacheStore()
	raw, _ := json.Marshal([]spec.ToolStatus{{Name: "x"}})
	if err := store.PutEntry("k", cache.Entry{Payload: raw, Resolved: time.Now().Add(-2 * probeCacheTTL)}); err != nil {
		t.Fatal(err)
	}
	if e, ok := store.Get("k"); !ok || e.FreshTTL(probeCacheTTL) {
		t.Fatal("an entry past the TTL must be stale")
	}
}
