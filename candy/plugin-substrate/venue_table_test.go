package substratekind

import (
	"testing"
)

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
		"android":     "parent",
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
