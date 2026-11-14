package substratekind

// kubevirt_body_test.go — the kubevirt BODY's own validation tests, moved here with the body
// (opencharly/spec#304 unit 6).
//
// They used to compile spec's shipped schema and look up #KubevirtValue; the def lives in THIS
// plugin's CUE module now, so the harness composes exactly what the host's gate composes — the
// spec base and this plugin's served unit — and asserts against THAT value.

import (
	"testing"

	"cuelang.org/go/cue"
)

// A realistic kind:kubevirt template (container_disk boot + cloud-init + a network + a GPU)
// passes the host-side value gate def.
func TestKubevirtValue_TemplateAccepted(t *testing.T) {
	body := `{
	cluster: "prod"
	namespace: "vms"
	source: {kind: "container_disk", image: "quay.io/example/vmbox:v1", pull_policy: "IfNotPresent"}
	memory: "4G"
	cpu: {cores: 2, sockets: 1}
	firmware: {bootloader: "efi", efi_secure_boot: false}
	devices: {rng: true, autoattach_serial_console: true}
	gpus: [{resource_name: "nvidia.com/gpu"}]
	network: {interface: "masquerade", model: "virtio", ports: [{name: "ssh", port: 22, protocol: "TCP"}]}
	run_strategy: "Always"
	cloud_init: {hostname: "kv1", package: ["sudo"]}
}`
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath("#KubevirtValue"))
	if !def.Exists() {
		t.Fatal("#KubevirtValue is not in the composed unit")
	}
	// compile in the def's OWN context — a value is only valid inside the context that built it
	// (the kindcluster helper's lesson; a fresh cuecontext.New() here silently breaks the unify).
	if err := def.Unify(v.Context().CompileString(body)).Validate(); err != nil {
		t.Fatalf("a valid kind:kubevirt template rejected by #KubevirtValue: %v", err)
	}
}

// The def is CLOSED (a typo is an error, not a silent drop) — the closedness the host value gate
// relies on.
func TestKubevirtValue_UnknownKeyRejected(t *testing.T) {
	body := `{source: {kind: "container_disk", image: "x"}, bogus_key: 1}`
	v := composedBodySchema(t)
	def := v.LookupPath(cue.ParsePath("#KubevirtValue"))
	if !def.Exists() {
		t.Fatal("#KubevirtValue is not in the composed unit")
	}
	if err := def.Unify(v.Context().CompileString(body)).Validate(); err == nil {
		t.Fatal("a kind:kubevirt template with an unknown key must fail the closed gate")
	}
}
