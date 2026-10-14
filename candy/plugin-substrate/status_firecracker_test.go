package substratekind

// status_firecracker_test.go — coverage for the firecracker word's registration and the status
// collector's logic. `collectFirecrackerStatus` itself needs the reverse-channel executor (to
// re-hydrate the resolved-project envelope), so the test exercises the legs it is built from —
// entry selection, the lease name, the image ref and the template resolution — plus the DECLARED
// TRAITS, which are the part a reader must not have to take on trust.

import (
	"testing"

	"github.com/opencharly/spec/spec"
)

// TestFirecrackerTraits asserts the two halves of the load-bearing decision: the venue is "ssh"
// (the contract's venue enum has no firecracker member, and a microVM really is reached over SSH),
// and NO exclusive venue is claimed — IsVmVenue is ssh AND exclusive_venue, so claiming one would
// misroute the deploy into `charly vm build`/`charly vm create`, the bug fe624e5 fixed for kubevirt.
// Flip either and this gate fails.
func TestFirecrackerTraits(t *testing.T) {
	tr, ok := substrateTraits["firecracker"]
	if !ok {
		t.Fatal("substrateTraits has no firecracker row: the word is registered but declares no behaviour")
	}

	if tr.Venue != "ssh" {
		t.Errorf("venue = %q, want \"ssh\" (reached over SSH; the venue enum has no firecracker member)", tr.Venue)
	}

	if tr.ExclusiveVenue {
		t.Error("ExclusiveVenue is set: IsVmVenue (ssh AND exclusive_venue) would then be true and the deploy would be misrouted into `charly vm build`/`charly vm create`")
	}

	if tr.MachineVenue {
		t.Error("MachineVenue is set: a microVM holds no host-global machine resource to arbitrate")
	}

	if !tr.ImageBacked {
		t.Error("ImageBacked is not set: the box IS an OCI image carrying the rootfs and kernel artefacts")
	}

	if !tr.BedTarget {
		t.Error("BedTarget is not set: a disposable bed must be able to boot a real microVM (WI-13)")
	}

	if !tr.SupportsEphemeral {
		t.Error("SupportsEphemeral is not set: a microVM is create/destroy-able per run, which is what the ephemeral TTL seam expects")
	}
}

// TestSubstrateWordsAllDeclareTraits is the invariant that keeps a half-registered word from
// shipping: every word this provider serves must carry a traits row, because the kernel reads the
// behaviour OFF THE TRAITS and a missing row means "external-in-place" by accident rather than by
// decision.
func TestSubstrateWordsAllDeclareTraits(t *testing.T) {
	found := false

	for _, w := range substrateWords {
		if w == "firecracker" {
			found = true
		}

		if _, ok := substrateTraits[w]; !ok {
			t.Errorf("substrateWords serves %q but substrateTraits declares no row for it", w)
		}
	}

	if !found {
		t.Fatalf("substrateWords = %v, want it to contain firecracker", substrateWords)
	}
}

// TestFirecrackerDeployEntries asserts the collector selects ONLY target:firecracker deploys — the
// gate that makes it emit a row for a microVM deploy and nothing for the others.
func TestFirecrackerDeployEntries(t *testing.T) {
	deploy := map[string]*spec.Deploy{
		"fc-lab":   {Target: "firecracker", From: "fc-host"},
		"a-pod":    {Target: "pod", Image: "somebox"},
		"a-kube":   {Target: "kubernetes", From: "prod"},
		"a-fc-two": {Target: "firecracker", From: "other"},
	}

	got := firecrackerDeployEntries(deploy)
	if len(got) != 2 || got[0] != "a-fc-two" || got[1] != "fc-lab" {
		t.Fatalf("firecrackerDeployEntries = %v, want the two firecracker entries sorted", got)
	}

	// A map with no firecracker entry selects nothing — which is what makes the collector's absence
	// visible rather than silent.
	if n := len(firecrackerDeployEntries(map[string]*spec.Deploy{"a-pod": {Target: "pod"}})); n != 0 {
		t.Fatalf("want 0 entries for a firecracker-less map, got %d", n)
	}
}

// TestFirecrackerLeaseName_Sanitized asserts the status row names the SAME sanitized lease the
// substrate will boot, so the row and the microVM agree.
func TestFirecrackerLeaseName_Sanitized(t *testing.T) {
	got := firecrackerLeaseName("check:firecracker/docker")
	if got == "" || got == "check:firecracker/docker" {
		t.Fatalf("lease name must be sanitized, got %q", got)
	}

	if plain := firecrackerLeaseName("fc-lab"); plain != "fc-lab" {
		t.Fatalf("plain name must be preserved, got %q", plain)
	}
}

// TestFirecrackerImageRef asserts the collector resolves the box a deploy boots: its explicit image
// (the OCI artefact), else the deploy name.
func TestFirecrackerImageRef(t *testing.T) {
	if got := firecrackerImageRef("fc-lab", &spec.Deploy{Image: "ghcr.io/opencharly/fc-runner:v1"}); got != "ghcr.io/opencharly/fc-runner:v1" {
		t.Errorf("explicit image: got %q", got)
	}

	if got := firecrackerImageRef("fc-lab", &spec.Deploy{}); got != "fc-lab" {
		t.Errorf("fallback image: got %q, want the deploy name", got)
	}
}

// TestFirecrackerSpecFor asserts the resolve leg's new arm: a referenced template resolves to an
// envelope carrying the RAW body (the venue is the deploy's own, so this leg invents no host), and
// the nil cases stay nil.
func TestFirecrackerSpecFor(t *testing.T) {
	body := []byte(`{"kernel":{"from":"ghcr.io/opencharly/firecracker-kernel:v0.1"},"containerd":{"snapshotter":"blockfile"}}`)
	templates := &spec.ProjectTemplates{Firecracker: map[string]spec.RawBody{"fc-host": body}}
	node := &spec.Deploy{Target: "firecracker", From: "fc-host"}

	got := firecrackerSpecFor(templates, node)
	if got == nil {
		t.Fatal("firecrackerSpecFor returned nil for a referenced template; the new resolve arm is not reached")
	}

	if len(got.Raw) == 0 {
		t.Error("the resolved envelope carries no raw body, so a consumer could not decode the profile")
	}

	if got.Host != "" {
		t.Errorf("the resolve leg invented host %q: the host is the DEPLOY's venue, not a template field", got.Host)
	}

	if firecrackerSpecFor(nil, node) != nil || firecrackerSpecFor(templates, nil) != nil {
		t.Error("nil templates or nil node must resolve to nil")
	}

	if firecrackerSpecFor(templates, &spec.Deploy{Target: "firecracker", From: "absent"}) != nil {
		t.Error("an unreferenced template must resolve to nil")
	}

	if firecrackerSpecFor(templates, &spec.Deploy{Target: "firecracker"}) != nil {
		t.Error("a node with no from: must resolve to nil")
	}
}
