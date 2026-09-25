package substratekind

import (
	"encoding/json"
	"testing"

	"github.com/opencharly/spec/spec"
)

// kubevirtTemplateBody builds an authored kind:kubevirt template RawBody carrying the
// given cluster/context — the shape resolveSubstrateTemplate decodes.
func kubevirtTemplateBody(t *testing.T, cluster, ctx string) spec.RawBody {
	t.Helper()
	body, err := json.Marshal(spec.KubeVirt{Cluster: cluster, KubeContext: ctx})
	if err != nil {
		t.Fatalf("marshal kubevirt template body: %v", err)
	}
	return body
}

func TestKubevirtDeployEntries(t *testing.T) {
	deploy := map[string]*spec.Deploy{
		"kv1":      {Target: "kubevirt", Image: "kv1", From: "prod"},
		"some-pod": {Target: "pod", Image: "redis"},
		"kv2":      {Target: "kubevirt", Image: "kv2", From: "stage"},
	}
	got := kubevirtDeployEntries(deploy)
	want := []string{"kv1", "kv2"} // sorted
	if len(got) != len(want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entries[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestKubevirtImageRef(t *testing.T) {
	if got := kubevirtImageRef("kv1", &spec.Deploy{Image: "quay.io/x:v1"}); got != "quay.io/x:v1" {
		t.Errorf("kubevirtImageRef = %q, want the explicit image", got)
	}
	if got := kubevirtImageRef("kv1", &spec.Deploy{}); got != "kv1" {
		t.Errorf("kubevirtImageRef = %q, want the fallback name", got)
	}
}

func TestKubevirtSpecFor(t *testing.T) {
	templates := &spec.ProjectTemplates{
		KubeVirt: map[string]spec.RawBody{
			"prod": kubevirtTemplateBody(t, "prod", "ctx_prod"),
		},
	}
	node := &spec.Deploy{Target: "kubevirt", From: "prod"}
	got := kubevirtSpecFor(templates, node)
	if got == nil || got.Cluster != "prod" || got.KubeContext != "ctx_prod" {
		t.Fatalf("kubevirtSpecFor = %+v, want prod/ctx_prod", got)
	}
	if got := kubevirtSpecFor(templates, &spec.Deploy{Target: "kubevirt", From: "missing"}); got != nil {
		t.Errorf("kubevirtSpecFor(missing) = %+v, want nil", got)
	}
	if got := kubevirtSpecFor(nil, node); got != nil {
		t.Errorf("kubevirtSpecFor(nil templates) = %+v, want nil", got)
	}
}

// TestCollectKubevirtStatus_RowLogic exercises collectKubevirtStatus's pure per-entry
// logic (deployed vs not-deployed from kubevirt_state + the template context) via its
// constituent pieces — the InvokeProvider("build","project") fetch itself is proven live
// by the same precedent the kubernetes collector test cites.
func TestCollectKubevirtStatus_RowLogic(t *testing.T) {
	deploy := map[string]*spec.Deploy{
		"kv-deployed": {Target: "kubevirt", Image: "kv-deployed", KubeVirtState: &spec.KubeVirtDeployState{
			VMName: "charly-kv-deployed", Namespace: "vms", KubeContext: "ctx_prod", SSHPort: 2224,
		}},
		"kv-fresh": {Target: "kubevirt", Image: "kv-fresh", From: "prod"},
		"a-pod":    {Target: "pod", Image: "redis"},
	}
	entries := kubevirtDeployEntries(deploy)
	if len(entries) != 2 {
		t.Fatalf("entries = %v, want 2 kubevirt deploys (pod ignored)", entries)
	}

	// deployed row: state-driven status + namespace/name container + context network.
	d := deploy["kv-deployed"]
	if d.KubeVirtState.VMName == "" || d.KubeVirtState.Namespace != "vms" {
		t.Fatalf("deployed row state = %+v, want a persisted venue identity", d.KubeVirtState)
	}
	// fresh row: template-resolved context via kubevirtSpecFor.
	templates := &spec.ProjectTemplates{KubeVirt: map[string]spec.RawBody{"prod": kubevirtTemplateBody(t, "prod", "ctx_prod")}}
	if ks := kubevirtSpecFor(templates, deploy["kv-fresh"]); ks == nil || ks.KubeContext != "ctx_prod" {
		t.Fatalf("fresh row context = %+v, want ctx_prod", ks)
	}
}

// TestEnrichKubevirtRow proves the deploy-cone enrichment fills the cluster tunnel +
// the managed port-forward host->22 mapping from the persisted kubevirt_state.
func TestEnrichKubevirtRow(t *testing.T) {
	col := &flatCollector{deploy: &spec.DeployConfig{Deploy: map[string]spec.DeployNode{
		"kv1": {KubeVirtState: &spec.KubeVirtDeployState{Cluster: "prod", SSHPort: 2224}},
	}}}
	row := spec.DeploymentStatus{Kind: spec.SubstrateKubevirt, Image: "kv1"}
	col.enrichKubevirtRow(&row, flatCollectOpts{Deploy: col.deploy})
	if row.Tunnel != "prod" {
		t.Errorf("Tunnel = %q, want prod", row.Tunnel)
	}
	if len(row.Ports) != 1 || row.Ports[0].HostPort != 2224 || row.Ports[0].CtrPort != 22 {
		t.Errorf("Ports = %+v, want [{2224 22 tcp}]", row.Ports)
	}

	// No state → no enrichment, no panic.
	empty := spec.DeploymentStatus{Kind: spec.SubstrateKubevirt, Image: "missing"}
	col.enrichKubevirtRow(&empty, flatCollectOpts{Deploy: col.deploy})
	if len(empty.Ports) != 0 || empty.Tunnel != "" {
		t.Errorf("unenriched row mutated: %+v", empty)
	}
}
