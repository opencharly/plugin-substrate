package substratekind

// status_firecracker.go — the firecracker substrate's OpStatus (mirrors status_kindcluster.go, R3).
//
// A `target: firecracker` deploy runs no container as a SERVICE on this host: it boots a microVM
// directly under KVM from a rootfs artefact plus a kernel, and is then addressed over SSH (its
// declared traits are venue "ssh" WITHOUT exclusive_venue — see plugin.go, where the absence is the
// load-bearing part). So this collector reports the DECLARED row — the host venue the microVMs run
// on, the box they boot, and the lease name the substrate will use — and never live guest health:
// a live probe is the substrate plugin's own conversation with the VMM and a check's business, not
// status's.
//
// Everything it needs is fetched from the host over the established InvokeProvider("build","project")
// seam (the same one status_kubernetes.go / status_kindcluster.go use), and resolving the referenced
// kind:firecracker template is an in-package call to this provider's own resolve leg (resolve.go),
// never a cross-plugin Invoke.

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/opencharly/sdk/deploykit"
	"github.com/opencharly/sdk/kit"
	"github.com/opencharly/spec/spec"
)

// collectFirecrackerStatus serves the firecracker substrate's OpStatusCollect: it re-hydrates the
// resolved-project envelope over the reverse channel, enumerates every declared target:firecracker
// deploy node, and emits one row per entry.
func collectFirecrackerStatus(ctx context.Context, req spec.SubstrateStatusRequest) (spec.SubstrateStatusReply, error) {
	rp, err := fetchResolvedProject(ctx)
	if err != nil {
		return spec.SubstrateStatusReply{}, fmt.Errorf("firecracker status-collect: %w", err)
	}

	entries := firecrackerDeployEntries(rp.Deploy)
	if len(entries) == 0 {
		return spec.SubstrateStatusReply{}, nil
	}

	rows := make([]spec.DeploymentStatus, 0, len(entries))

	for _, name := range entries {
		node := rp.Deploy[name]
		row := spec.DeploymentStatus{
			Kind:     spec.SubstrateFirecracker,
			Source:   "tree",
			Image:    firecrackerImageRef(name, node),
			Instance: firecrackerLeaseName(name),
			RunMode:  req.RunMode,
		}

		// The HOST the microVMs run on: the resolved template's own host when one is filled (the
		// venue resolver's job — this plugin does not invent it), else the node's from: ref, which
		// at least names the profile the deploy boots from.
		if fc := firecrackerSpecFor(rp.Templates, node); fc != nil && fc.Host != "" {
			row.Network = fc.Host
		} else if node != nil && node.From != "" {
			row.Network = node.From
		}

		// charly addresses nothing inside the guest until a check does, so the row reports what
		// charly KNOWS rather than a health it has not probed — "declared" mirrors the kindcluster
		// and kubernetes collectors, which report the declared shape without a live probe.
		row.Status = "declared"

		rows = append(rows, row)
	}

	return spec.SubstrateStatusReply{Rows: rows}, nil
}

// firecrackerDeployEntries returns the names of every target:firecracker deploy in the
// resolved-project's folded Deploy map, in deterministic (sorted) order.
func firecrackerDeployEntries(deploy map[string]*spec.Deploy) []string {
	if len(deploy) == 0 {
		return nil
	}

	var names []string

	for name, node := range deploy {
		if deploykit.ClassifyTarget(node) == "firecracker" {
			names = append(names, name)
		}
	}

	sort.Strings(names)

	return names
}

// firecrackerImageRef resolves the box a firecracker deploy boots: the node's explicit image (the
// OCI artefact carrying the rootfs and the kernel), else the deploy name.
func firecrackerImageRef(name string, node *spec.Deploy) string {
	if node != nil && node.Image != "" {
		return node.Image
	}

	return name
}

// firecrackerLeaseName is the per-lease microVM name a deploy will create. It mirrors the
// deploy-side derivation (the same sanitisation the kindcluster arm uses) so the status row names
// the SAME lease the substrate will boot, rather than the raw deploy key.
func firecrackerLeaseName(name string) string {
	return kit.SanitizeDeployName(name)
}

// firecrackerSpecFor resolves the kind:firecracker template referenced by node.From against the
// resolved-project's firecracker template bodies. Nil when unreferenced or absent. Uses this SAME
// provider's own template-resolve leg (resolve.go) — an in-package call, never a cross-plugin Invoke.
func firecrackerSpecFor(templates *spec.ProjectTemplates, node *spec.Deploy) *spec.ResolvedFirecracker {
	if templates == nil || node == nil || node.From == "" {
		return nil
	}

	body, ok := templates.Firecracker[node.From]
	if !ok {
		return nil
	}

	out, err := resolveSubstrateTemplate(spec.SubstrateTemplateResolveRequest{
		Firecracker: &spec.FirecrackerResolveInput{Firecracker: body},
	})
	if err != nil {
		return nil
	}

	var reply spec.FirecrackerResolveReply
	if err := json.Unmarshal(out, &reply); err != nil {
		return nil
	}

	return reply.Resolved
}
