package substratekind

// firecracker_capability_test.go — the DECLARATION half of the firecracker kind-body move
// (opencharly/spec#304 unit 7), and the test round 1's review showed missing: the two-arm corpus
// sweep proves PARITY between the base's twin and the composed unit, but it cannot tell whether
// this plugin DECLARES the gate def at all — if the InputDef were absent, arm B would silently
// fall back to the base's twin and the sweep would still print "firecracker differs in zero
// bodies".
//
// This test fails on an ASSERTION when the declaration is missing, and resolves the def out of
// the composed unit — so it also proves the body actually arrived in this module.

import (
	"context"
	"testing"

	"cuelang.org/go/cue"

	pb "github.com/opencharly/spec/proto"
)

func TestFirecrackerCapabilityDeclaresItsGateDef(t *testing.T) {
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	var fc *pb.ProvidedCapability
	for _, c := range caps.GetProvided() {
		if c.GetClass() == "kind" && c.GetWord() == "firecracker" {
			fc = c
		}
	}
	if fc == nil {
		t.Fatal("no class:kind capability serves the firecracker word")
	}
	if got := fc.GetInputDef(); got != "#FirecrackerValue" {
		t.Fatalf("kind:firecracker InputDef = %q, want %q — the body lives in this plugin's CUE module (schema/firecracker.cue), so the host must be told which def gates it", got, "#FirecrackerValue")
	}

	// and the def it names really resolves out of the composed unit.
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath(fc.GetInputDef()))
	if !def.Exists() {
		t.Fatalf("%s does not resolve in the composed unit — the body did not arrive", fc.GetInputDef())
	}
}
