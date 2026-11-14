package substratekind

// kindcluster_body_test.go — the kindcluster BODY's own validation tests, moved here with the
// body (opencharly/spec#304 unit 5).
//
// They used to compile spec's shipped schema and look up #Kindcluster / #KindclusterValue; the
// defs live in THIS plugin's CUE module now, so the harness composes exactly what the host's
// gate composes — the spec base and this plugin's served unit — and asserts against THAT value.
// Moving the tests with the body is what keeps "the body moved and its contract still holds"
// from becoming an assertion: the same five properties are checked, at the home they moved to.

import (
	"context"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	pb "github.com/opencharly/spec/proto"
	sdkschema "github.com/opencharly/spec/schema"
	"github.com/opencharly/spec/schemacompose"
)

// composedBodySchema returns base & served-unit — the value the host's kind-value gate reads
// a moved body def out of.
func composedBodySchema(t *testing.T) cue.Value {
	t.Helper()
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	ctx := cuecontext.New()
	unit, err := schemacompose.ServedText(ctx, caps.GetSchemaCue())
	if err != nil {
		t.Fatalf("served schema does not compose: %v", err)
	}
	base, err := schemacompose.ServedSchema(ctx, sdkschema.FS, ".")
	if err != nil {
		t.Fatalf("spec base schema: %v", err)
	}
	merged := base.Unify(unit)
	if merged.Err() != nil {
		t.Fatalf("base & plugin does not unify: %v", merged.Err())
	}
	return merged
}

// kindclusterUnifies unifies #Kindcluster with the given literal (nil = accepted).
func kindclusterUnifies(t *testing.T, literal string) error {
	t.Helper()
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath("#Kindcluster"))
	if !def.Exists() {
		t.Fatal("#Kindcluster is not in the composed unit — the body is missing from this plugin's schema directory")
	}
	return def.Unify(v.Context().CompileString(literal)).Validate(cue.Concrete(false))
}

// kindclusterValueUnifies unifies #KindclusterValue with the given literal.
func kindclusterValueUnifies(t *testing.T, literal string) error {
	t.Helper()
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath("#KindclusterValue"))
	if !def.Exists() {
		t.Fatal("#KindclusterValue is not in the composed unit")
	}
	return def.Unify(v.Context().CompileString(literal)).Validate(cue.Concrete(false))
}

// A cluster-policy-only template (box empty) with every kind-specific knob set is accepted —
// the shape a `kindcluster:` TEMPLATE node authors.
func TestKindclusterAcceptsPolicyTemplate(t *testing.T) {
	literal := `{
		box: ""
		engine: "podman"
		node_image: "kindest/node:v1.37.0@sha256:a1ed56cfb0e7b93589bdf97c8cd566405a265939e3620fc4f5de89adff580ae5"
		nodes: [
			{role: "control-plane", extra_port_mappings: [{container_port: 30080, host_port: 30080}]},
			{role: "worker"},
		]
		kubeconfig_context: "kind-kind"
		default_namespace: "apps"
		admission_policy: "restricted"
	}`
	if err := kindclusterUnifies(t, literal); err != nil {
		t.Fatalf("valid kindcluster template rejected: %v", err)
	}
}

// #KindclusterValue accepts the DEPLOY shape too (a `from:` cross-ref), routed by shape in
// the loader.
func TestKindclusterValueAcceptsDeployShape(t *testing.T) {
	if err := kindclusterValueUnifies(t, `{from: "kindcluster-base", engine: "docker"}`); err != nil {
		t.Fatalf("kindcluster deploy shape rejected: %v", err)
	}
}

// An unknown key is rejected (the def is CLOSED).
func TestKindclusterRejectsUnknownKey(t *testing.T) {
	if err := kindclusterUnifies(t, `{box: "", bogus_key: true}`); err == nil {
		t.Fatal("kindcluster with an unknown key must be rejected")
	}
}

// The engine vocabulary is closed, and `nerdctl` is the DELETED engine word: charly's word is
// containerd, which the plugin maps to kind's own nerdctl provider at the edge.
func TestKindclusterEngineVocabulary(t *testing.T) {
	if err := kindclusterUnifies(t, `{box: "", engine: "nerdctl"}`); err == nil {
		t.Fatal("kindcluster accepts engine: nerdctl; the engine word is deleted")
	}
	if err := kindclusterUnifies(t, `{box: "", engine: "containerd"}`); err != nil {
		t.Fatalf("kindcluster rejects engine: containerd: %v", err)
	}
}

// The cluster-policy sub-blocks are the SAME defs the kubernetes body uses, so a
// kubernetes-style policy block is accepted verbatim.
func TestKindclusterAcceptsKubernetesPolicyBlocks(t *testing.T) {
	literal := `{
		box: ""
		storage: {class_default: "standard", access_mode_default: "ReadWriteOnce"}
		ingress: {enabled: true, class: "nginx"}
		image_default: {pull_policy: "IfNotPresent"}
		pod_default: {node_selector: {disk: "ssd"}}
		defaults: {labels: {"managed-by": "opencharly"}}
	}`
	if err := kindclusterUnifies(t, literal); err != nil {
		t.Fatalf("kindcluster with kubernetes policy blocks rejected: %v", err)
	}
}
