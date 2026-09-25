package substratekind

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/opencharly/sdk/deploykit"
	"github.com/opencharly/spec/spec"
)

// status_kubevirt.go — the KubeVirt substrate's OpStatus. A kind:kubevirt deploy
// runs a cluster-scheduled VM; this collector reports the DEPLOY-TREE state — which
// kind:kubevirt nodes exist, their cluster/context, and the persisted kubevirt_state
// venue identity — resolved from the same resolved-project envelope the kubernetes
// collector uses (R3). It performs NO live cluster probe: the LIVE VirtualMachine
// state is a `kubevirt:` check (candy/plugin-kubevirt), not a `charly status` row,
// exactly as the kubernetes collector reports generation state while `kube:` probes
// the live cluster. The flat fan-out's enrichKubevirtRow applies the deploy-config
// detail (mirroring status_vm.go's own split).

// collectKubevirtStatus serves the kubevirt substrate's OpStatusCollect.
func collectKubevirtStatus(ctx context.Context, req spec.SubstrateStatusRequest) (spec.SubstrateStatusReply, error) {
	rp, err := fetchResolvedProject(ctx)
	if err != nil {
		return spec.SubstrateStatusReply{}, fmt.Errorf("kubevirt status-collect: %w", err)
	}
	return spec.SubstrateStatusReply{Rows: buildKubevirtRows(rp, req.RunMode)}, nil
}

// buildKubevirtRows is the PURE row-builder the collector delegates to (so it is
// directly unit-testable without the reverse channel): one row per declared
// kind:kubevirt deploy, with Status from the persisted kubevirt_state when present
// ("deployed") or "not-deployed" otherwise, and the cluster/context resolved from
// kubevirt_state or the referenced kind:kubevirt template.
func buildKubevirtRows(rp *spec.ResolvedProject, runMode string) []spec.DeploymentStatus {
	if rp == nil {
		return nil
	}
	entries := kubevirtDeployEntries(rp.Deploy)
	if len(entries) == 0 {
		return nil
	}
	rows := make([]spec.DeploymentStatus, 0, len(entries))
	for _, name := range entries {
		node := rp.Deploy[name]
		row := spec.DeploymentStatus{
			Kind:      spec.SubstrateKubevirt,
			Source:    "cluster",
			Image:     kubevirtImageRef(name, node),
			Container: name,
			RunMode:   runMode,
		}
		// The persisted venue identity (kubevirt_state) is the deploy-tree truth:
		// the cluster/context the VM landed in + the VM name (domain-scoped). A
		// deploy with no state has never been applied → "not-deployed".
		if node != nil && node.KubeVirtState != nil && node.KubeVirtState.VMName != "" {
			row.Status = "deployed"
			row.Network = node.KubeVirtState.KubeContext
			if node.KubeVirtState.Namespace != "" {
				row.Container = node.KubeVirtState.Namespace + "/" + node.KubeVirtState.VMName
			}
		} else {
			row.Status = "not-deployed"
			if kv := kubevirtSpecFor(rp.Templates, node); kv != nil && kv.KubeContext != "" {
				row.Network = kv.KubeContext
			} else if node != nil && node.From != "" {
				row.Network = node.From
			}
		}
		rows = append(rows, row)
	}
	return rows
}

// kubevirtDeployEntries returns the names of every kind:kubevirt deploy in the
// resolved-project's folded Deploy map, in deterministic (sorted) order.
func kubevirtDeployEntries(deploy map[string]*spec.Deploy) []string {
	if len(deploy) == 0 {
		return nil
	}
	var names []string
	for name, node := range deploy {
		if deploykit.ClassifyTarget(node) == "kubevirt" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// kubevirtImageRef resolves the image a kubevirt deploy runs: the node's explicit
// Image, falling back to the deploy name (mirrors kubernetesImageRef).
func kubevirtImageRef(name string, node *spec.Deploy) string {
	if node != nil && node.Image != "" {
		return node.Image
	}
	return name
}

// kubevirtSpecFor resolves the kind:kubevirt template referenced by node.From
// against the resolved-project's kubevirt template bodies, via this SAME
// provider's own template-resolve leg (resolve.go) — an in-package call.
func kubevirtSpecFor(templates *spec.ProjectTemplates, node *spec.Deploy) *spec.ResolvedKubeVirt {
	if templates == nil || node == nil || node.From == "" {
		return nil
	}
	body, ok := templates.KubeVirt[node.From]
	if !ok {
		return nil
	}
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{KubeVirt: &spec.KubeVirtResolveInput{KubeVirt: body}})
	if err != nil {
		return nil
	}
	var reply spec.KubeVirtResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		return nil
	}
	return reply.Resolved
}
