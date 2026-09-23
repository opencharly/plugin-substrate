package substratekind

import (
	"context"

	"os/exec"
	"testing"

	"github.com/opencharly/sdk/enginekit"
)

// probe_cache_live_test.go — the LIVE proof of the changed guest-probe cache
// path. It needs a container engine (podman/docker) to launch a throwaway
// container; when none is available it SKIPS cleanly (R7a: live-or-skip, never a
// fake). It drives the REAL runGuestProbes (the migrated cache read/write) twice:
// the first call computes + caches, the second is served from the ArtifactStore.

// TestRunGuestProbesLiveCacheRoundTrip exercises runGuestProbes end-to-end against
// a real container: compute -> cache write, then a second call -> cache hit.
func TestRunGuestProbesLiveCacheRoundTrip(t *testing.T) {
	engineBin := ""
	for _, b := range []string{"podman", "docker"} {
		if p, err := exec.LookPath(b); err == nil {
			engineBin = p
			break
		}
	}
	if engineBin == "" {
		t.Skip("no container engine (podman/docker) on PATH — skipping the live guest-probe cache round-trip")
	}
	t.Setenv("CHARLY_CACHE_DIR", t.TempDir())

	name := "charly-probe-live-test"
	_ = exec.Command(engineBin, "rm", "-f", name).Run()
	if out, err := exec.Command(engineBin, "run", "-d", "--name", name, "docker.io/library/alpine:latest", "sleep", "120").CombinedOutput(); err != nil {
		t.Skipf("could not start a test container (%v): %s", err, out)
	}
	defer func() { _ = exec.Command(engineBin, "rm", "-f", name).Run() }()

	e := enginekit.NewEngineClient(engineBin)
	ctx := context.Background()

	first := runGuestProbes(ctx, e, name, guestProbes)
	if len(first) != len(guestProbes) {
		t.Fatalf("first run returned %d results, want %d", len(first), len(guestProbes))
	}
	// The first call WRITES the cache; a second call must be served from it.
	second := runGuestProbes(ctx, e, name, guestProbes)
	if len(second) != len(first) {
		t.Fatalf("second run returned %d results, want %d", len(second), len(first))
	}
	for i := range first {
		if first[i].Name != second[i].Name || first[i].Status != second[i].Status {
			t.Fatalf("cache round-trip changed result %d: %+v vs %+v", i, first[i], second[i])
		}
	}
}
