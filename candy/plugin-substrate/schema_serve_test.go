package substratekind

import (
	"context"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"

	pb "github.com/opencharly/spec/proto"
	sdkschema "github.com/opencharly/spec/schema"
	"github.com/opencharly/spec/schemacompose"
)

// TestNewMetaServesNonEmptySchema pins the uniform plugin contract: NewMeta's
// Describe reply MUST carry this plugin's own non-empty CUE schema (schema_cue).
// It FAILS without the change — before it, NewMeta passed a nil schema FS and
// served an EMPTY schema ("there is no schema-less plugin").
func TestNewMetaServesNonEmptySchema(t *testing.T) {
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if strings.TrimSpace(caps.GetSchemaCue()) == "" {
		t.Fatal("Describe served an EMPTY schema_cue; every plugin MUST ship a non-empty CUE schema")
	}
	if !strings.Contains(caps.GetSchemaCue(), "#SubstratePlugin") {
		t.Fatalf("served schema does not carry #SubstratePlugin:\n%s", caps.GetSchemaCue())
	}
}

// TestServedSchemaSplicesOntoHostBase exercises the SDK/host load gate this change
// exists to satisfy: it composes the plugin's served schema_cue EXACTLY as both
// sides of the wire compose it — spec/schemacompose on the serve side
// (sdk.BuildCapabilitiesWithRequires → ServedSchema) and on the host side
// (registerPluginUnitSchema → ServedText) — and then reproduces charly's
// `compileBasePlusServed` splice (`base.Unify(unit)`) against the real spec base
// schema. A schema that is empty, will not compose, or will not splice onto the
// base is rejected here exactly as at the host load gate.
//
// The pre-compose form of this test concatenated the base body and the served text
// and called CompileString on the result. That is no longer faithful — the served
// body now carries `import "github.com/opencharly/spec/schema"` (schema/pod.cue), and
// a bare CompileString has no build instance to resolve an import through. Composing
// each side as a module instance is what the runtime does, so it is what the test
// must do.
func TestServedSchemaSplicesOntoHostBase(t *testing.T) {
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	served := caps.GetSchemaCue()
	if strings.TrimSpace(served) == "" {
		t.Fatal("empty schema_cue: cannot splice onto the host base")
	}
	ctx := cuecontext.New()
	// The host side: one module instance per unit (ServedText), then one unify.
	unit, err := schemacompose.ServedText(ctx, served)
	if err != nil {
		t.Fatalf("served schema does not compose on its own (a plugin schema is ONE CUE module package): %v", err)
	}
	// The base side: the spec module's own `schema` package, in the SAME context (a
	// cue.Value is only valid inside the context that built it, so the unify below
	// would be meaningless across two).
	base, err := schemacompose.ServedSchema(ctx, sdkschema.FS, ".")
	if err != nil {
		t.Fatalf("spec base schema: %v", err)
	}
	if merged := base.Unify(unit); merged.Err() != nil {
		t.Fatalf("served schema does not splice onto the base (base & plugin): %v", merged.Err())
	}
}

// TestKindPodCapabilityDeclaresItsGateDef pins the DECLARATION half of the kind-body
// move (opencharly/spec#304 unit 3): the `kind: pod` capability must name the
// `#<Word>Value` body def it owns, because the host resolves a kind's gate def FROM
// THE DECLARATION (kindValueDefPath) and reads it out of the composed unit. Without
// this declaration the host keeps answering from the base's spec.KindValueDefs
// compatibility fallback, so the def could not be deleted from the base at all.
//
// It FAILS on an ASSERTION without the change (the capability declares no InputDef).
func TestKindPodCapabilityDeclaresItsGateDef(t *testing.T) {
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	var pod *pb.ProvidedCapability
	for _, c := range caps.GetProvided() {
		if c.GetClass() == "kind" && c.GetWord() == "pod" {
			pod = c
		}
	}
	if pod == nil {
		t.Fatal("no class:kind capability serves the pod word")
	}
	if got := pod.GetInputDef(); got != "#PodValue" {
		t.Fatalf("kind:pod InputDef = %q, want %q — the body lives in this plugin's CUE module (schema/pod.cue), so the host must be told which def gates it", got, "#PodValue")
	}
	// The routing is UNCHANGED by the declaration: pod still advertises its substrate traits,
	// so it still folds through the host-pre-decoded substrate path rather than the flat
	// plugin-input path.
	if pod.GetDeployTraits() == nil {
		t.Fatal("kind:pod declares no DeployTraits — the word must stay a deploy substrate")
	}
	if !pod.GetStructural() {
		t.Fatal("kind:pod must stay Structural: the host pre-decodes its canonical node and the plugin echoes it")
	}
}

// TestServedSchemaDeclaresThePodKindBody pins that this plugin SERVES the pod kind body
// and that the served #PodValue actually judges: a pod-shaped body is accepted, a typo'd
// field is rejected by the CLOSED def, and a deploy-shaped body still passes through the
// contract's #DeployValue arm. Both halves run against the COMPOSED value the host reads
// (base & unit), never against a file on disk, because the composed value is the surface
// the gate consults.
//
// The reject case is the R7 witness: an OPEN #Pod (opened with `...`) makes the typo
// acceptable and this test RED on an assertion, which is exactly the closedness the move
// must preserve.
func TestServedSchemaDeclaresThePodKindBody(t *testing.T) {
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
	composed := base.Unify(unit)
	if composed.Err() != nil {
		t.Fatalf("base & plugin: %v", composed.Err())
	}
	def := composed.LookupPath(cue.ParsePath("#PodValue"))
	if err := def.Err(); err != nil {
		t.Fatalf("the composed unit declares no #PodValue — the pod kind body is not served: %v", err)
	}

	judge := func(label, body string) error {
		v, perr := parser.ParseFile("probe", body)
		if perr != nil {
			t.Fatalf("%s: parse the probe body: %v", label, perr)
		}
		return def.Unify(ctx.BuildFile(v)).Validate()
	}

	// ACCEPT: a pod body with a box + a plan, and one with the sidecar/env_default fields.
	if err := judge("pod body", "{box: \"fedora\", plan: [{run: \"true\"}]}"); err != nil {
		t.Errorf("a pod-shaped body must be ACCEPTED by #PodValue: %v", err)
	}
	if err := judge("pod sidecar body", "{sidecar: [{sidecar: {\"db\": {image: \"redis\"}}}], env_default: {A: 1}}"); err != nil {
		t.Errorf("a pod body carrying sidecar/env_default must be ACCEPTED by #PodValue: %v", err)
	}
	// ACCEPT: the contract's deploy arm — a `pod:` node whose body has the deploy shape.
	if err := judge("deploy-shaped body", "{from: \"app\", image: \"app\"}"); err != nil {
		t.Errorf("a deploy-shaped pod body must be ACCEPTED through #DeployValue: %v", err)
	}
	// REJECT: #Pod is CLOSED, so a typo'd field cannot slip through.
	if err := judge("typo'd body", "{box: \"fedora\", bogus: 1}"); err == nil {
		t.Error("#PodValue ACCEPTED a typo'd pod field — #Pod is no longer closed")
	}
}

// TestKindLocalCapabilityDeclaresItsGateDef pins that the `local` kind's capability declares
// the gate def its body now lives behind. opencharly/spec#304 unit 4 moved the `kind: local`
// body into this plugin's CUE module (schema/local.cue), so the host resolves #LocalValue
// from this DECLARATION and reads it out of the composed unit. Without this declaration the
// host keeps answering from the base's spec.KindValueDefs compatibility fallback, so the def
// could not be deleted from the base at all.
//
// It FAILS on an ASSERTION without the change (the capability declares no InputDef).
func TestKindLocalCapabilityDeclaresItsGateDef(t *testing.T) {
	caps, err := NewMeta().Describe(context.Background(), &pb.Empty{})
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	var local *pb.ProvidedCapability
	for _, c := range caps.GetProvided() {
		if c.GetClass() == "kind" && c.GetWord() == "local" {
			local = c
		}
	}
	if local == nil {
		t.Fatal("no class:kind capability serves the local word")
	}
	if got := local.GetInputDef(); got != "#LocalValue" {
		t.Fatalf("kind:local InputDef = %q, want %q — the body lives in this plugin's CUE module (schema/local.cue), so the host must be told which def gates it", got, "#LocalValue")
	}
	// The routing is UNCHANGED by the declaration: local still advertises its substrate
	// traits, so it still folds through the host-pre-decoded substrate path.
	if local.GetDeployTraits() == nil {
		t.Fatal("kind:local declares no DeployTraits — the word must stay a deploy substrate")
	}
	if !local.GetStructural() {
		t.Fatal("kind:local must stay Structural: the host pre-decodes its canonical node and the plugin echoes it")
	}
}

// TestServedSchemaDeclaresTheLocalKindBody pins that this plugin SERVES the local kind body
// and that the served #LocalValue actually judges: a local body is accepted, a typo'd field
// is rejected by the CLOSED def, the moved def still forbids a PATH override, and a
// deploy-shaped body still passes through the contract's #DeployValue arm. Both halves run
// against the COMPOSED value the host reads (base & unit), never against a file on disk,
// because the composed value is the surface the gate consults.
//
// The R7 witness for THIS leg is the typo case: opening the moved #Local with `...` opens
// the COMPOSED def too, so a typo'd field is accepted and this test goes RED on an
// assertion. The PATH case is a CHAIN guard rather than a leg-1 witness, stated honestly:
// until the contract leg drops `spec/schema/local.cue`, the base's `#Local` still forbids
// the override, so no leg-1 plant can make it red — it becomes load-bearing the moment that
// leg lands, which is exactly when the moved body is the only #Local left.
func TestServedSchemaDeclaresTheLocalKindBody(t *testing.T) {
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
	composed := base.Unify(unit)
	if composed.Err() != nil {
		t.Fatalf("base & plugin: %v", composed.Err())
	}
	def := composed.LookupPath(cue.ParsePath("#LocalValue"))
	if err := def.Err(); err != nil {
		t.Fatalf("the composed unit declares no #LocalValue — the local kind body is not served: %v", err)
	}

	judge := func(label, body string) error {
		v, perr := parser.ParseFile("probe", body)
		if perr != nil {
			t.Fatalf("%s: parse the probe body: %v", label, perr)
		}
		return def.Unify(ctx.BuildFile(v)).Validate()
	}

	// ACCEPT: a local body with a candy stack, a description and a plan.
	if err := judge("local body", `{candy: ["shell"], description: "a host", plan: [{run: "true"}]}`); err != nil {
		t.Errorf("a local-shaped body must be ACCEPTED by #LocalValue: %v", err)
	}
	// ACCEPT: the empty stack the schema documents as a staged name reservation.
	if err := judge("empty stack", `{candy: []}`); err != nil {
		t.Errorf("an empty candy stack must be ACCEPTED by #LocalValue: %v", err)
	}
	// ACCEPT: the contract's deploy arm — a `local:` node whose body has the deploy shape.
	if err := judge("deploy-shaped body", `{from: "app", image: "app"}`); err != nil {
		t.Errorf("a deploy-shaped local body must be ACCEPTED through #DeployValue: %v", err)
	}
	// REJECT: #Local is CLOSED, so a typo'd field cannot slip through.
	if err := judge("typo'd body", `{candy: ["shell"], bogus: 1}`); err == nil {
		t.Error("#LocalValue ACCEPTED a typo'd local field — #Local is no longer closed")
	}
	// REJECT: the moved def still forbids overriding PATH.
	if err := judge("PATH override", `{candy: ["shell"], env: {PATH: "/usr/bin"}}`); err == nil {
		t.Error("#LocalValue ACCEPTED an env.PATH override — the moved #Local lost its PATH constraint")
	}
}
