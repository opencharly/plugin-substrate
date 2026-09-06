package substratekind

// load_pod_substrate_test.go — the Cutover C settled-contract repro for the substrate
// structural kind OpLoad (spec v0.2026249.2215 + sdk v0.2026249.2239): the canonical
// imageless agent_provisioned iterate-entity pod (agent_provisioned: true + an iterate:
// block and NO image:) must PARSE, thread through the host-pre-decoded StandaloneLoad
// channel, and SURVIVE the substrate OpLoad echo — not only the legacy imageful spelling.
//
// THE SEMANTIC BREAK this file pins: under the pre-wave embedded contract
// (spec v0.2026241.1322 at tag v2026.242.0533), the pod body failed the substrate
// load/validate chain — the iterate: block was misclassified as an in-substrate member
// (kind-word key agent: inside its value mapping) and the imageless shape failed the
// box-required validator — so the node VANISHED from the resolved fleet and every
// downstream consumer saw "no entity check-agent-live" (the charly-cli + distro-arch
// check-agent-live repro). sdk #221/#225 fixed the parse (a declared #Deploy body field
// is DATA — the member scan never looks inside its value) and spec #105/#107 removed
// group:, added the DeployDeclaredFields channel, and exempted AgentProvisioned from
// ValidateDeployRequiresBox.
//
// This file MIRRORS sdk loaderkit/parse_guard_test.go's TestParse_SubstrateDeclaredIterateStaysData
// and plugin-build's TestBuildProjectEnvelope_ImagelessAgentProvisionedPodSurvives at the
// SUBSTRATE-KIND level, over the SAME exported primitives the load chain runs (loaderkit
// parse + CUE gates → spec.ValidateDeploymentTree → the StandaloneLoad env thread → the
// provider's own Invoke/OpLoad echo the host folds into uf.Fleet):
//
//   1. the imageless agent_provisioned pod + iterate parses (CUE #NodeDoc + step gates),
//     validates into the Fleet (the AgentProvisioned exemption), and the OpLoad echo
//     round-trips agent_provisioned + the iterate data byte-faithfully;
//   2. the imageful pod still parses and echoes (the legacy spelling is unchanged);
//   3. the discriminator: the same imageless node WITHOUT the flag is still rejected
//     by the Fleet gate (the gate did not go away — the exemption is the only change);
//   4. the template-shape echo stays byte-faithful for the same canonical body.

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/opencharly/sdk"
	"github.com/opencharly/sdk/loaderkit"
	pb "github.com/opencharly/spec/proto"
	"github.com/opencharly/spec/spec"
	"gopkg.in/yaml.v3"
)

// liveThreaded models the LIVE host threading for the canonical charly-cli iterate-entity
// bed shape — the registry-derived snapshot the loader-threaded leg feeds (sdk
// loaderkit/parse_guard_test.go's livePodThreaded, mirrored verbatim): pod is BOTH a
// deploy substrate AND a structural kind, agent is a threaded kind word colliding with
// values inside a declared #Deploy field's body, and the deploy-declared-fields channel
// carries the #Deploy body field names the registered substrate schema declares. Host-fed
// DATA, not this provider's own vocabulary.
var liveThreaded = spec.Threaded{
	Kinds:            map[string]bool{"pod": true, "agent": true, "check": true},
	DeploySubstrates: map[string]bool{"pod": true, "vm": true},
	StructuralKinds:  map[string]bool{"pod": true, "vm": true, "agent": true},
	DeployDeclaredFields: map[string]map[string]bool{
		"pod": {
			"from": true, "image": true, "env": true, "disposable": true,
			"plan": true, "iterate": true, "record": true, "instrument": true,
			"cpus": true, "ram": true, "disk_size": true, "snapshot": true,
			"update_gate": true, "agent_provisioned": true,
		},
	},
}

// docNode parses raw YAML into the document node ParseDoc consumes.
func docNode(t *testing.T, s string) *yaml.Node {
	t.Helper()
	var n yaml.Node
	if err := yaml.Unmarshal([]byte(s), &n); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	return &n
}

// bodyMap decodes an opaque body JSON for assertions.
func bodyMap(t *testing.T, body json.RawMessage) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("body json: %v", err)
	}
	return m
}

// opLoadEcho drives the provider's OWN Invoke OpLoad arm with a host-threaded
// StandaloneLoad env and returns the echoed result JSON — the exact call the host's
// registry-backed dispatch makes for a substrate kind load.
func opLoadEcho(t *testing.T, word string, env *spec.StructuralKindLoadEnv) []byte {
	t.Helper()
	envJSON, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal load env: %v", err)
	}
	reply, err := provider{}.Invoke(context.Background(), &pb.InvokeRequest{
		Op:       sdk.OpLoad,
		Reserved: word,
		EnvJson:  envJSON,
	})
	if err != nil {
		t.Fatalf("Invoke(OpLoad %s): %v", word, err)
	}
	return reply.GetResultJson()
}

const imagelessAgentProvisionedDoc = `
check-agent-live:
  pod:
    agent_provisioned: true
    iterate:
      sandbox: check-agent-pod
      agent: [check-agent-live-claude]
      plateau_iteration: 1
      prompt: Reply with one short acknowledgement.
      note: false
      env: {}
  watcher:
    pod:
      image: watcher-img
`

// TestSubstrateOpLoad_ImagelessAgentProvisionedPodSurvives is the semantic repro: the NEW
// canonical imageless agent_provisioned iterate-entity pod parses, passes BOTH CUE gates
// the load chain runs (the per-entity #NodeDoc structural gate and the step-typing gate),
// survives the Fleet gate (the ValidateDeployRequiresBox AgentProvisioned exemption), and
// the substrate OpLoad echo hands the host back its agent_provisioned flag + iterate data
// byte-faithfully. Under the pre-wave embedded contract this shape aborted the
// load/validate chain, so the node never reached uf.Fleet and every downstream consumer
// saw "no entity check-agent-live".
func TestSubstrateOpLoad_ImagelessAgentProvisionedPodSurvives(t *testing.T) {
	if err := loaderkit.ValidateNodeDocCUE("repro-imageless", []byte(imagelessAgentProvisionedDoc)); err != nil {
		t.Fatalf("ValidateNodeDocCUE: the settled pod schema must accept the imageless agent_provisioned shape: %v", err)
	}
	if err := loaderkit.ValidateNodeFormSteps("repro-imageless", []byte(imagelessAgentProvisionedDoc), liveThreaded, loaderkit.DocParser{}); err != nil {
		t.Fatalf("ValidateNodeFormSteps: %v", err)
	}
	_, pp, err := loaderkit.ParseDoc(docNode(t, imagelessAgentProvisionedDoc), liveThreaded)
	if err != nil {
		t.Fatalf("ParseDoc: %v", err)
	}
	if len(pp.Nodes) != 1 {
		t.Fatalf("nodes = %d, want 1 — the imageless pod must survive the parse", len(pp.Nodes))
	}
	pn := pp.Nodes[0]
	if pn.Disc != "pod" {
		t.Fatalf("disc = %q, want pod", pn.Disc)
	}
	// iterate is DATA, never a member: the only member is the deploy-level sibling.
	if len(pn.Children) != 1 || pn.Children[0].Name != "watcher" {
		t.Fatalf("children = %+v, want just the deploy-level sibling watcher", pn.Children)
	}

	// The host pre-decodes the canonical node and threads it as the deploy shape; model
	// that channel from the parsed body and drive the provider's own OpLoad echo.
	var dep spec.Deploy
	if err := json.Unmarshal(pn.Body, &dep); err != nil {
		t.Fatalf("decode canonical body into spec.Deploy: %v", err)
	}
	fleet := map[string]spec.Deploy{"check-agent-live": dep}
	if err := spec.ValidateDeploymentTree(fleet); err != nil {
		t.Fatalf("ValidateDeploymentTree: the imageless agent_provisioned pod must stay in the Fleet: %v", err)
	}

	echo := opLoadEcho(t, "pod", &spec.StructuralKindLoadEnv{
		Standalone: &spec.StandaloneLoad{Shape: "deploy", Deploy: &dep},
	})
	var echoed spec.Deploy
	if err := json.Unmarshal(echo, &echoed); err != nil {
		t.Fatalf("decode echo into spec.Deploy: %v", err)
	}
	if !echoed.AgentProvisioned {
		t.Fatalf("echo.AgentProvisioned = false, want true (the flag must survive the echo)")
	}
	if echoed.Iterate == nil || echoed.Iterate.Sandbox != "check-agent-pod" {
		t.Fatalf("echo.Iterate = %+v, want the iterate data intact", echoed.Iterate)
	}
	// iterate stayed DATA: the pod body carries NO members — the watcher deploy-level
	// sibling is folded as its own fleet root (FleetWalkPreOrder: a deploy-level member
	// is a folded top-level entry walked as its own root, never inside its owner).
	if echoed.HasMembers() {
		t.Fatalf("echo member tree = %+v, want NO members (iterate is data; the watcher sibling folds as its own root)", echoed.Member)
	}

	// Byte-faithfulness of the echo round-trip (RDD: a canonical spec.Deploy round-trips
	// through JSON byte-faithfully — the echo IS the former in-proc decode; both sides
	// marshal the SAME spec.Deploy struct, so the bytes must match exactly).
	remarshaled, merr := json.Marshal(dep)
	if merr != nil {
		t.Fatalf("remarshal dep: %v", merr)
	}
	if string(echo) != string(remarshaled) {
		t.Fatalf("echo not byte-faithful:\n echo: %s\n want: %s", echo, remarshaled)
	}
}

// TestSubstrateOpLoad_ImagefulPodStillParses mirrors sdk's
// TestParse_SubstrateDeclaredIterateStaysData verbatim at the substrate-kind level: the
// LEGACY imageful spelling keeps parsing and echoing — the declared-fields channel
// excludes declared fields from the member scan, nothing else moved.
func TestSubstrateOpLoad_ImagefulPodStillParses(t *testing.T) {
	doc := `
check-agent-live:
  pod:
    image: sandbox-img
    iterate:
      sandbox: check-agent-pod
      agent: [check-agent-live-claude]
      plateau_iteration: 1
      prompt: Reply with one short acknowledgement.
      note: false
      env: {}
  watcher:
    pod:
      image: watcher-img
`
	if err := loaderkit.ValidateNodeDocCUE("repro-imageful", []byte(doc)); err != nil {
		t.Fatalf("ValidateNodeDocCUE: %v", err)
	}
	if err := loaderkit.ValidateNodeFormSteps("repro-imageful", []byte(doc), liveThreaded, loaderkit.DocParser{}); err != nil {
		t.Fatalf("ValidateNodeFormSteps: %v", err)
	}
	_, pp, err := loaderkit.ParseDoc(docNode(t, doc), liveThreaded)
	if err != nil {
		t.Fatalf("ParseDoc: %v", err)
	}
	if len(pp.Nodes) != 1 || pp.Nodes[0].Disc != "pod" {
		t.Fatalf("nodes = %+v, want one pod node", pp.Nodes)
	}
	pn := pp.Nodes[0]
	body := bodyMap(t, pn.Body)
	if body["image"] != "sandbox-img" {
		t.Fatalf("body.image = %v, want sandbox-img", body["image"])
	}
	if len(pn.Children) != 1 || pn.Children[0].Name != "watcher" {
		t.Fatalf("children = %+v, want just the deploy-level sibling watcher", pn.Children)
	}

	var dep spec.Deploy
	if err := json.Unmarshal(pn.Body, &dep); err != nil {
		t.Fatalf("decode canonical body: %v", err)
	}
	echo := opLoadEcho(t, "pod", &spec.StructuralKindLoadEnv{
		Standalone: &spec.StandaloneLoad{Shape: "deploy", Deploy: &dep},
	})
	remarshaled, merr := json.Marshal(dep)
	if merr != nil {
		t.Fatalf("remarshal dep: %v", merr)
	}
	if string(echo) != string(remarshaled) {
		t.Fatalf("echo not byte-faithful:\n echo: %s\n want: %s", echo, remarshaled)
	}
}

// TestSubstrateOpLoad_PlainImagelessPodStillRejected is the discriminator: the Fleet gate
// did NOT go away — a plain imageless pod without the AgentProvisioned flag is still
// rejected with the box-required error; the exemption is the only change.
func TestSubstrateOpLoad_PlainImagelessPodStillRejected(t *testing.T) {
	fleet := map[string]spec.Deploy{
		"bare-pod": {Target: "pod"},
	}
	err := spec.ValidateDeploymentTree(fleet)
	if err == nil {
		t.Fatal("ValidateDeploymentTree: want the box-required rejection for a plain imageless pod")
	}
	if want := "lacks required"; !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want it to contain %q", err, want)
	}
}

// TestSubstrateOpLoad_TemplateEchoByteFaithful pins the template-shape arm: the host
// pre-decoded typed template value is echoed VERBATIM (the host folds it into uf.Pod).
func TestSubstrateOpLoad_TemplateEchoByteFaithful(t *testing.T) {
	tmpl, err := json.Marshal(spec.Pod{Box: "sandbox-img", EnvDefaults: map[string]string{"PORT": "8080"}})
	if err != nil {
		t.Fatalf("marshal template: %v", err)
	}
	echo := opLoadEcho(t, "pod", &spec.StructuralKindLoadEnv{
		Standalone: &spec.StandaloneLoad{Shape: "template", Template: tmpl},
	})
	if string(echo) != string(tmpl) {
		t.Fatalf("template echo not byte-faithful:\n echo: %s\n tmpl: %s", echo, tmpl)
	}
}
