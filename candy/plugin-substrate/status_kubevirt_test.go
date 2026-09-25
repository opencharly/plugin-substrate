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

// TestBuildKubevirtRows exercises the collector's PURE row-builder directly (the
// collector body delegates to it): deployed vs not-deployed, the namespace/name
// container, and the context resolution. This is the collector function under test,
// not its constituents.
func TestBuildKubevirtRows(t *testing.T) {
	tmpl := kubevirtTemplateBody(t, "prod", "ctx_prod")
	rp := &spec.ResolvedProject{
		Deploy: map[string]*spec.Deploy{
			"kv-deployed": {Target: "kubevirt", Image: "kv-deployed", KubeVirtState: &spec.KubeVirtDeployState{
				VMName: "charly-kv-deployed", Namespace: "vms", KubeContext: "ctx_prod", SSHPort: 2224,
			}},
			"kv-fresh": {Target: "kubevirt", Image: "kv-fresh", From: "prod"},
			"a-pod":    {Target: "pod", Image: "redis"},
		},
		Templates: &spec.ProjectTemplates{KubeVirt: map[string]spec.RawBody{"prod": tmpl}},
	}
	rows := buildKubevirtRows(rp, "quadlet")
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (pod ignored): %+v", len(rows), rows)
	}
	byImage := map[string]spec.DeploymentStatus{}
	for _, r := range rows {
		byImage[r.Image] = r
	}
	dep := byImage["kv-deployed"]
	if dep.Status != "deployed" || dep.Container != "vms/charly-kv-deployed" || dep.Network != "ctx_prod" {
		t.Errorf("deployed row = %+v, want deployed/vms:charly-kv-deployed/ctx_prod", dep)
	}
	fresh := byImage["kv-fresh"]
	if fresh.Status != "not-deployed" || fresh.Network != "ctx_prod" {
		t.Errorf("fresh row = %+v, want not-deployed/ctx_prod (from the template)", fresh)
	}
	if buildKubevirtRows(nil, "") != nil {
		t.Error("nil ResolvedProject must yield no rows")
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
