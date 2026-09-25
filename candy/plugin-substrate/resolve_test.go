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
