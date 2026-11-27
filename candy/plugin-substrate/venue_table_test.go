package substratekind

import (
	"slices"
	"testing"
)

// TestSubstrateWordsExactSet pins the SERVED kind word set — the surface the host resolves a
// `kind:` node through — and requires substrateTraits to carry a row for exactly those words.
// A word retired from the contract must leave the provider in the same cutover, and one that
// leaves must not leave its traits row behind: the kernel reads deploy behaviour off the
// traits, so an orphan row is a declaration for a word nothing serves.
func TestSubstrateWordsExactSet(t *testing.T) {
	want := []string{"pod", "vm", "kubernetes", "local", "kubevirt", "kindcluster", "firecracker"}
	got := slices.Clone(substrateWords)
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("substrateWords = %v, want exactly %v (the substrate words the contract still declares)", got, want)
	}
	for word := range substrateTraits {
		if !slices.Contains(substrateWords, word) {
			t.Errorf("substrateTraits declares a row for %q, which substrateWords does not serve", word)
		}
	}
}

// TestSubstrateTraits_VenueTable pins the DECLARED venue for every substrate word — the
// single source the kernel stamps onto node.Descent (kit.StampDescent) and every
// venue-consult site reads. The kubevirt row in particular must carry its OWN venue
// ("kubevirt"), NOT "ssh": it is a cluster-managed VirtualMachine CR whose lifecycle an
// out-of-process plugin owns, so a bare `Venue == "ssh"` test in a consumer must name the
// host-libvirt vm and nothing else (R1 — the venue≠substrate defect).
func TestSubstrateTraits_VenueTable(t *testing.T) {
	want := map[string]string{
		"pod":         "container",
		"vm":          "ssh",
		"kubevirt":    "kubevirt",
		"local":       "shell",
		"kubernetes":  "shell",
		"kindcluster": "none",
	}
	for word, venue := range want {
		tr, ok := substrateTraits[word]
		if !ok {
			t.Errorf("substrateTraits has no %q row", word)
			continue
		}
		if tr.Venue != venue {
			t.Errorf("substrateTraits[%q].Venue = %q, want %q", word, tr.Venue, venue)
		}
	}
	// The vm and kubevirt venues must be DISTINCT — the whole point of the split.
	if substrateTraits["vm"].Venue == substrateTraits["kubevirt"].Venue {
		t.Fatal("vm and kubevirt must not share a venue value (kubevirt is plugin-owned, not host-libvirt)")
	}
}
