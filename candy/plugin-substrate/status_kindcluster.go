package substratekind

// status_kindcluster.go — the kindcluster substrate's OpStatus (mirrors
// status_kubernetes.go, R3). A `target: kindcluster` deploy does not run a
// container as a SERVICE on this host — it provisions a local Kubernetes-in-Docker
// cluster (via the upstream `kind` tool) on the operator's container engine and is
// then addressed by kubeconfig context. So this collector reports PROVISIONING
// state (the declared cluster + its resolved kubeconfig context + engine), never
// live pod health (that is a `kube:` check, candy/plugin-kube). Every input it
// needs (the folded project deploy tree, the kindcluster template bodies) is
// fetched from the host via the established InvokeProvider("build","project")
// seam — the SAME one status_kubernetes.go uses — and resolving the referenced
// kindcluster template itself is an in-package call to this provider's own resolve
// leg (resolve.go), never a cross-plugin Invoke.

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/opencharly/sdk/deploykit"
	"github.com/opencharly/sdk/kit"
	"github.com/opencharly/spec/spec"
)

// collectKindclusterStatus serves the kindcluster substrate's OpStatusCollect. It
// re-hydrates the resolved-project envelope over the reverse channel, enumerates
// every declared target:kindcluster deploy node, and emits one row per entry.
func collectKindclusterStatus(ctx context.Context, req spec.SubstrateStatusRequest) (spec.SubstrateStatusReply, error) {
	rp, err := fetchResolvedProject(ctx)
	if err != nil {
		return spec.SubstrateStatusReply{}, fmt.Errorf("kindcluster status-collect: %w", err)
	}

	entries := kindclusterDeployEntries(rp.Deploy)
	if len(entries) == 0 {
		return spec.SubstrateStatusReply{}, nil
	}

	rows := make([]spec.DeploymentStatus, 0, len(entries))
	for _, name := range entries {
		node := rp.Deploy[name]
		row := spec.DeploymentStatus{
			Kind:      spec.SubstrateKindcluster,
			Source:    "tree",
			Image:     kindclusterImageRef(name, node),
			Container: kindclusterClusterName(name),
			RunMode:   req.RunMode,
		}

		// The kubeconfig context the deploy addresses: the resolved template's own
		// kubeconfig_context, else the node's from: ref (the resolved cluster name is
		// derived from the deployment name — see candy/plugin-kube's kindcluster.go).
		if kc := kindclusterSpecFor(rp.Templates, node); kc != nil && kc.KubeconfigContext != "" {
			row.Network = kc.KubeconfigContext
		} else if node != nil && node.From != "" {
			row.Network = node.From
		}

		// kind manages its own cluster lifecycle outside charly's container store, so
		// the collector reports the DECLARED row (a live `kube:` check is the health
		// proof); "declared" mirrors kubernetes's "not-generated" reporting of what
		// charly knows without a live probe.
		row.Status = "declared"

		rows = append(rows, row)
	}
	return spec.SubstrateStatusReply{Rows: rows}, nil
}

// kindclusterDeployEntries returns the names of every target:kindcluster deploy in
// the resolved-project's folded Deploy map, in deterministic (sorted) order.
func kindclusterDeployEntries(deploy map[string]*spec.Deploy) []string {
	if len(deploy) == 0 {
		return nil
	}
	var names []string
	for name, node := range deploy {
		if deploykit.ClassifyTarget(node) == "kindcluster" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// kindclusterImageRef resolves the image a kindcluster deploy runs, mirroring the
// kubernetes collector: the node's explicit Box (carried as Image), else the
// deploy name.
func kindclusterImageRef(name string, node *spec.Deploy) string {
	if node != nil && node.Image != "" {
		return node.Image
	}
	return name
}

// kindclusterClusterName is the kind cluster name a deploy provisions. It mirrors
// the deploy preresolver's derivation — candy/plugin-kube's kindcluster.go
// `kindClusterName` calls `kit.SanitizeDeployName` on the deploy name — so the
// status row names the SAME cluster the create leg does.
func kindclusterClusterName(name string) string {
	return kit.SanitizeDeployName(name)
}

// kindclusterSpecFor resolves the kind:kindcluster template referenced by node.From
// against the resolved-project's kindcluster template bodies. Nil when unreferenced
// or absent. Uses this SAME provider's own template-resolve leg (resolve.go) — an
// in-package call, never a cross-plugin Invoke.
func kindclusterSpecFor(templates *spec.ProjectTemplates, node *spec.Deploy) *spec.ResolvedKindcluster {
	if templates == nil || node == nil || node.From == "" {
		return nil
	}
	body, ok := templates.Kindcluster[node.From]
	if !ok {
		return nil
	}
	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{Kindcluster: &spec.KindclusterResolveInput{Kindcluster: body}})
	if err != nil {
		return nil
	}
	var reply spec.KindclusterResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		return nil
	}
	return reply.Resolved
}
