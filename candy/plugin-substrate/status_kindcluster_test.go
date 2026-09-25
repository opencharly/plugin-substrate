package substratekind

// status_kindcluster_test.go — coverage for the kindcluster status collector's
// logic. collectKindclusterStatus itself needs the reverse-channel executor (to
// re-hydrate the resolved-project envelope), so the test exercises the pure legs it
// is built from — entry selection, the row's cluster name, and the template
// resolution — which is exactly the logic that would silently no-op without the
// collector (a fixture with no target:kindcluster deploy selects NO entries).

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/opencharly/spec/spec"
)

// TestKindclusterDeployEntries asserts the collector selects ONLY target:kindcluster
// deploys — the gate that makes it emit a row for a kindcluster deploy and nothing
// for others.
func TestKindclusterDeployEntries(t *testing.T) {
	deploy := map[string]*spec.Deploy{
		"kind-lab":   {Target: "kindcluster", From: "kind-lab-cluster"},
		"a-pod":      {Target: "pod", Image: "somebox"},
		"a-kube":     {Target: "kubernetes", From: "prod"},
		"a-kind-two": {Target: "kindcluster", From: "other"},
	}
	got := kindclusterDeployEntries(deploy)
	if len(got) != 2 || got[0] != "a-kind-two" || got[1] != "kind-lab" {
		t.Fatalf("kindclusterDeployEntries = %v, want the two kindcluster entries sorted", got)
	}
	// A deploy map with no kindcluster entry selects nothing — the no-op the
	// collector must not be (i.e. this is what makes the collector's absence visible).
	if n := len(kindclusterDeployEntries(map[string]*spec.Deploy{"a-pod": {Target: "pod"}})); n != 0 {
		t.Fatalf("want 0 entries for a kindcluster-less map, got %d", n)
	}
}

// TestKindclusterClusterName_Sanitized asserts the status row names the SAME
// sanitized cluster the deploy preresolver creates (kit.SanitizeDeployName), so the
// row and the live cluster agree.
func TestKindclusterClusterName_Sanitized(t *testing.T) {
	got := kindclusterClusterName("check:kindcluster/docker")
	if got == "" || got == "check:kindcluster/docker" {
		t.Fatalf("cluster name must be sanitized, got %q", got)
	}
	// A plain name is unchanged.
	if plain := kindclusterClusterName("kind-lab"); plain != "kind-lab" {
		t.Fatalf("plain name must be preserved, got %q", plain)
	}
}

// TestKindclusterImageRef asserts the collector resolves the image a kindcluster
// deploy runs (its explicit Box, else the deploy name).
func TestKindclusterImageRef(t *testing.T) {
	if got := kindclusterImageRef("kind-lab", &spec.Deploy{Image: "app"}); got != "app" {
		t.Errorf("explicit image: got %q", got)
	}
	if got := kindclusterImageRef("kind-lab", &spec.Deploy{}); got != "kind-lab" {
		t.Errorf("fallback to name: got %q", got)
	}
}

// TestEphemeralKindclusterLiveness_NoClusterIsDead asserts the reap-orphans
// kindcluster probe's contract against the real `kind` binary when one is present:
// a cluster name that does not exist is reported DEAD (so its ephemeral is reaped).
// When `kind` is unavailable the probe is conservative (alive) — the safety
// property (never reap a cluster we cannot confirm is gone) — and the test asserts
// that direction instead. Either way the test fails if the kindcluster arm is
// removed (the switch would fall through to the default false/alive).
func TestEphemeralKindclusterLiveness_NoClusterIsDead(t *testing.T) {
	node := spec.Deploy{Target: "kindcluster", Engine: "podman"}
	alive := ephemeralUnderlyingResourceAlive(context.Background(), nil, "definitely-not-a-real-cluster", node)
	if _, lookErr := exec.LookPath("kind"); lookErr == nil {
		// kind is present: the absent cluster MUST be reported dead (alive=false).
		if alive {
			t.Fatal("with kind available, an absent cluster must be reported dead (alive=false)")
		}
		return
	}
	// kind absent: the probe must be conservative (alive=true) — never reap blind.
	if !alive {
		t.Fatal("without kind, the probe must be conservative (alive=true), never reap a possibly-live cluster")
	}
}

// kindcluster template to its kubeconfig context via this provider's own resolve arm.
func TestKindclusterSpecFor_ResolvesContext(t *testing.T) {
	body, err := json.Marshal(spec.Kindcluster{KubeconfigContext: "kind-lab"})
	if err != nil {
		t.Fatal(err)
	}
	templates := &spec.ProjectTemplates{
		Kindcluster: map[string]spec.RawBody{"kind-lab-cluster": body},
	}
	node := &spec.Deploy{Target: "kindcluster", From: "kind-lab-cluster"}
	res := kindclusterSpecFor(templates, node)
	if res == nil || res.KubeconfigContext != "kind-lab" {
		t.Fatalf("kindcluster template not resolved: %+v", res)
	}
	// No from: → nil (no context to report).
	if res := kindclusterSpecFor(templates, &spec.Deploy{Target: "kindcluster"}); res != nil {
		t.Fatalf("a from-less node must resolve to nil, got %+v", res)
	}
}
