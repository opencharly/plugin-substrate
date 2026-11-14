package substratekind

import (
	"encoding/json"
	"testing"

	"github.com/opencharly/spec/spec"
)

// TestResolveVm_FieldCopy covers the vm substrate-value de-type (Cutover L): OpResolve
// projects spec.Vm → ResolvedVm (field-copy + Raw). Without the resolve leg the kernel's
// vm build/deploy consumers cannot read the de-typed, opaque vm template.
func TestResolveVm_FieldCopy(t *testing.T) {
	body, err := json.Marshal(spec.Vm{Backend: "libvirt", Cpus: 4, Ram: "8G", Firmware: "uefi"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{
		Vm: &spec.VmResolveInput{Vm: body},
	})
	if err != nil {
		t.Fatalf("resolveSubstrateTemplate(vm): %v", err)
	}
	var reply spec.VmResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		t.Fatal(err)
	}
	r := reply.Resolved
	if r == nil || r.Backend != "libvirt" || r.Cpus != 4 || r.Ram != "8G" || r.Firmware != "uefi" {
		t.Fatalf("vm field copy failed: %+v", r)
	}
	if string(r.Raw) != string(body) {
		t.Errorf("Raw not preserved through resolve")
	}
}

// TestResolveKindcluster_FieldCopy covers the kindcluster substrate-value de-type:
// OpResolve projects spec.Kindcluster → ResolvedKindcluster (kubeconfig_context +
// Raw). Without the resolve leg a `from:`-referenced kindcluster deploy cannot
// resolve its cluster template.
func TestResolveKindcluster_FieldCopy(t *testing.T) {
	body, err := json.Marshal(spec.Kindcluster{
		Box:               "",
		Engine:            "podman",
		KubeconfigContext: "kind-lab",
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{
		Kindcluster: &spec.KindclusterResolveInput{Kindcluster: body},
	})
	if err != nil {
		t.Fatalf("resolveSubstrateTemplate(kindcluster): %v", err)
	}
	var reply spec.KindclusterResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		t.Fatal(err)
	}
	r := reply.Resolved
	if r == nil || r.KubeconfigContext != "kind-lab" {
		t.Fatalf("kindcluster field copy failed: %+v", r)
	}
	if string(r.Raw) != string(body) {
		t.Errorf("Raw not preserved through resolve")
	}
}

// TestResolveKubeVirt_FieldCopy covers the kubevirt substrate-value de-type: OpResolve
// projects spec.KubeVirt → ResolvedKubeVirt (cluster/context/namespace + Raw). Without
// this arm the kind:kubevirt template cannot be de-typed for the deploy consumer.
func TestResolveKubeVirt_FieldCopy(t *testing.T) {
	body, err := json.Marshal(spec.KubeVirt{Cluster: "prod", KubeContext: "ctx", Namespace: "vms"})
	if err != nil {
		t.Fatal(err)
	}
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{
		KubeVirt: &spec.KubeVirtResolveInput{KubeVirt: body},
	})
	if err != nil {
		t.Fatalf("resolveSubstrateTemplate(kubevirt): %v", err)
	}
	var reply spec.KubeVirtResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		t.Fatal(err)
	}
	r := reply.Resolved
	if r == nil || r.Cluster != "prod" || r.KubeContext != "ctx" || r.Namespace != "vms" {
		t.Fatalf("kubevirt field copy failed: %+v", r)
	}
	if string(r.Raw) != string(body) {
		t.Errorf("Raw not preserved through resolve")
	}
}

// TestResolvePod_Projection covers the pod substrate-value de-type: OpResolve projects an
// authored `pod:` template body straight into the CONTRACT's #ResolvedPod envelope
// (box/sidecar/secret/env_default/plan + Raw preserved).
//
// The pod body has no plugin-owned Go type any more (opencharly/spec#304 unit 3 moved
// #Pod into this plugin's CUE module and deleted spec.Pod), so this arm decodes the
// authored bytes into the envelope the KERNEL reads. The test pins that projection at its
// actual boundary: a field the envelope does not carry is dropped exactly as the former
// `spec.Pod` intermediate dropped it, and Raw keeps the authored bytes verbatim.
func TestResolvePod_Projection(t *testing.T) {
	body := []byte(`{"box":"fedora","sidecar":[{"sidecar":{"db":{"image":"redis"}}}],"env_default":{"A":"1"},"plan":[{"run":"true"}],"bogus":1}`)
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{
		Pod: &spec.PodResolveInput{Pod: body},
	})
	if err != nil {
		t.Fatalf("resolveSubstrateTemplate(pod): %v", err)
	}
	var reply spec.PodResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		t.Fatal(err)
	}
	r := reply.Resolved
	if r == nil {
		t.Fatal("pod resolve produced no envelope")
	}
	if r.Box != "fedora" {
		t.Errorf("Box = %q, want %q", r.Box, "fedora")
	}
	if len(r.Sidecar) != 1 || len(r.Sidecar[0].Sidecar) != 1 {
		t.Errorf("Sidecar = %+v, want the one authored wrapper", r.Sidecar)
	}
	if got := r.EnvDefaults["A"]; got != "1" {
		t.Errorf("EnvDefaults = %v, want A=1", r.EnvDefaults)
	}
	if len(r.Plan) != 1 {
		t.Errorf("Plan = %+v, want the one authored step", r.Plan)
	}
	if string(r.Raw) != string(body) {
		t.Errorf("Raw not preserved through resolve")
	}
}

// TestResolveLocal_Projection pins the OpResolve leg's local arm AFTER the body moved
// (opencharly/spec#304 unit 4): the authored bytes decode STRAIGHT into the contract's
// #ResolvedLocal envelope, with no plugin-side Go type of its own and Raw preserved. The
// `bogus` key is deliberately present — OpResolve is a projection, not the validator; the
// closedness of #Local is the host gate's job (schema_serve_test.go proves that half).
func TestResolveLocal_Projection(t *testing.T) {
	body := []byte(`{"candy":["shell"],"description":"a host","plan":[{"run":"true"}],"bogus":1}`)
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{
		Local: &spec.LocalResolveInput{Local: body},
	})
	if err != nil {
		t.Fatalf("resolveSubstrateTemplate(local): %v", err)
	}
	var reply spec.LocalResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		t.Fatal(err)
	}
	r := reply.Resolved
	if r == nil {
		t.Fatal("local resolve produced no envelope")
	}
	if len(r.Candy) != 1 || r.Candy[0] != "shell" {
		t.Errorf("Candy = %v, want [shell]", r.Candy)
	}
	if r.Description != "a host" {
		t.Errorf("Description = %q, want %q", r.Description, "a host")
	}
	if len(r.Plan) != 1 {
		t.Errorf("Plan = %+v, want the one authored step", r.Plan)
	}
	if string(r.Raw) != string(body) {
		t.Errorf("Raw not preserved through resolve")
	}
}

// TestResolveKubernetes_Projection and TestResolveKindcluster_Projection pin the two OpResolve
// arms opencharly/spec#304 unit 5 changed: the authored bodies live in THIS plugin's CUE module
// now, so each arm decodes the authored bytes straight into the contract's resolved envelope
// instead of through a spec-authored Go type. The point of the pair is the FIELD: the old code
// copied `KubeconfigContext` by hand, and the direct decode must arrive at the same value
// (both envelope fields carry the authored `kubeconfig_context` json tag) with Raw verbatim.
func TestResolveKubernetes_Projection(t *testing.T) {
	body := []byte(`{"box":"app","kubeconfig_context":"prod-cluster","kubeconfig":"/tmp/kc","default_namespace":"myns"}`)
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{
		Kubernetes: &spec.KubernetesResolveInput{Kubernetes: body},
	})
	if err != nil {
		t.Fatalf("resolveSubstrateTemplate(kubernetes): %v", err)
	}
	var reply spec.KubernetesResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		t.Fatal(err)
	}
	r := reply.Resolved
	if r == nil {
		t.Fatal("kubernetes resolve produced no envelope")
	}
	if r.KubeconfigContext != "prod-cluster" {
		t.Errorf("KubeconfigContext = %q, want %q", r.KubeconfigContext, "prod-cluster")
	}
	if string(r.Raw) != string(body) {
		t.Errorf("Raw not preserved through resolve")
	}
}

func TestResolveKindcluster_Projection(t *testing.T) {
	body := []byte(`{"kubeconfig_context":"kind-prod","engine":"docker"}`)
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{
		Kindcluster: &spec.KindclusterResolveInput{Kindcluster: body},
	})
	if err != nil {
		t.Fatalf("resolveSubstrateTemplate(kindcluster): %v", err)
	}
	var reply spec.KindclusterResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		t.Fatal(err)
	}
	r := reply.Resolved
	if r == nil {
		t.Fatal("kindcluster resolve produced no envelope")
	}
	if r.KubeconfigContext != "kind-prod" {
		t.Errorf("KubeconfigContext = %q, want %q", r.KubeconfigContext, "kind-prod")
	}
	if string(r.Raw) != string(body) {
		t.Errorf("Raw not preserved through resolve")
	}
}

// TestResolveKubeVirt_Projection pins the arm opencharly/spec#304 unit 6 rewrote. The authored
// kubevirt body lives in this plugin's CUE module now, so the arm decodes the authored bytes
// straight into the contract's resolved envelope instead of through a spec-authored Go type.
// The FIELD is the point: the old code copied Cluster/KubeContext/Namespace by hand, and the
// direct decode must arrive at the same three values (the envelope's json tags are the authored
// keys) with Raw verbatim.
func TestResolveKubeVirt_Projection(t *testing.T) {
	body := []byte(`{"cluster":"prod-cluster","kube_context":"ctx-1","namespace":"vms","memory":"4Gi"}`)
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{
		KubeVirt: &spec.KubeVirtResolveInput{KubeVirt: body},
	})
	if err != nil {
		t.Fatalf("resolveSubstrateTemplate(kubevirt): %v", err)
	}
	var reply spec.KubeVirtResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		t.Fatal(err)
	}
	r := reply.Resolved
	if r == nil {
		t.Fatal("kubevirt resolve produced no envelope")
	}
	if r.Cluster != "prod-cluster" || r.KubeContext != "ctx-1" || r.Namespace != "vms" {
		t.Errorf("projection lost the authored fields: cluster=%q context=%q ns=%q", r.Cluster, r.KubeContext, r.Namespace)
	}
	if string(r.Raw) != string(body) {
		t.Errorf("Raw not preserved through resolve")
	}
}

// TestResolveFirecracker_Projection exercises the arm opencharly/spec#304 unit 7 rewrote. The
// authored body lives in this plugin's CUE module now, and the envelope carries the RAW bytes
// alone (a firecracker template describes the HOST; the venue fills Host/SSHUser), so the arm is
// the pure "wrap the bytes" path — which is exactly what this test pins: the reply's Resolved is
// non-nil and carries the authored bytes verbatim, with no decode that could drop a field.
func TestResolveFirecracker_Projection(t *testing.T) {
	body := []byte(`{"description":"a microvm host","machine":{"vcpu_count":2},"kernel":{"path":"/boot/vmlinux"},"pool":[{"name":"ci"}]}`)
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{
		Firecracker: &spec.FirecrackerResolveInput{Firecracker: body},
	})
	if err != nil {
		t.Fatalf("resolveSubstrateTemplate(firecracker): %v", err)
	}
	var reply spec.FirecrackerResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		t.Fatal(err)
	}
	if reply.Resolved == nil {
		t.Fatal("firecracker resolve produced no envelope")
	}
	if string(reply.Resolved.Raw) != string(body) {
		t.Errorf("the envelope did not carry the authored bytes verbatim: %q", reply.Resolved.Raw)
	}
}
