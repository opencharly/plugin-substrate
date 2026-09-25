package substratekind

import (
	"encoding/json"
	"fmt"

	"github.com/opencharly/spec/spec"
)

// validate_kubevirt.go — the "kubevirt" capability's deep OpValidate check
// (ProvidedCapability.Validates=true). The host's value gate is closedness-only by
// design (charly/provider_kind_invoke.go documents why), so the XOR rules a CUE
// closed struct cannot express live HERE, in the kind's own plugin — the SAME
// split validate_vm.go documents for the PCI-hostdev concreteness.
//
// Enforced:
//   - source.kind is exactly one of the four arms (container_disk / data_volume /
//     pvc / clone). CUE pins the arm's fields but a Go decode is what catches an
//     EMPTY or unknown kind (the closedness gate sees no concrete value to check).
//   - source.kind == container_disk forbids source.storage_class (a containerDisk
//     is not a PVC-backed volume).
//   - each gpus[] entry sets EXACTLY ONE of resource_name / device_name.
//   - instancetype XOR explicit cpu (a matcher + an inline domain CPU conflict).
type kubevirtValidateBody struct {
	// Source is a POINTER so the deploy shape (no `source:` block — it carries
	// `from:`/`image:` instead) is distinguishable from a template that AUTHORS an
	// empty/partial source. Source-level rules apply to the TEMPLATE shape only.
	Source *struct {
		Kind         string `json:"kind"`
		Image        string `json:"image"`
		StorageClass string `json:"storage_class"`
		DataVolume   any    `json:"data_volume"`
		PVC          string `json:"pvc"`
		Clone        any    `json:"clone"`
	} `json:"source"`
	CPU struct {
		Cores   int `json:"cores"`
		Sockets int `json:"sockets"`
		Threads int `json:"threads"`
	} `json:"cpu"`
	Instancetype string `json:"instancetype"`
	GPUs         []struct {
		ResourceName string `json:"resource_name"`
		DeviceName   string `json:"device_name"`
	} `json:"gpus"`
}

var kubevirtSourceKinds = map[string]bool{
	"container_disk": true,
	"data_volume":    true,
	"pvc":            true,
	"clone":          true,
}

// validateKubevirtDeep runs the kubevirt kind's deep OpValidate check against the
// raw authored entity body the host threads via op.Params. Source-level rules apply
// ONLY when a `source:` block is authored (the TEMPLATE shape); the DEPLOY shape
// (`from:`/`image:`, no source) is skipped for those, matching validateVmDeep's arm
// selectivity.
func validateKubevirtDeep(paramsJSON json.RawMessage) (spec.Diagnostics, error) {
	var body kubevirtValidateBody
	if len(paramsJSON) > 0 {
		if err := json.Unmarshal(paramsJSON, &body); err != nil {
			return spec.Diagnostics{}, fmt.Errorf("plugin-substrate: kubevirt OpValidate: decode entity: %w", err)
		}
	}

	var diags spec.Diagnostics
	if body.Source != nil {
		switch {
		case body.Source.Kind == "":
			diags.Items = append(diags.Items, spec.Diagnostic{
				Severity: "error",
				Path:     "source.kind",
				Message:  "must be one of container_disk | data_volume | pvc | clone",
			})
		case !kubevirtSourceKinds[body.Source.Kind]:
			diags.Items = append(diags.Items, spec.Diagnostic{
				Severity: "error",
				Path:     "source.kind",
				Message:  fmt.Sprintf("%q is not a known kubevirt source kind (one of container_disk | data_volume | pvc | clone)", body.Source.Kind),
			})
		}
		// container_disk forbids storage_class (a containerDisk is not PVC-backed).
		if body.Source.Kind == "container_disk" && body.Source.StorageClass != "" {
			diags.Items = append(diags.Items, spec.Diagnostic{
				Severity: "error",
				Path:     "source.storage_class",
				Message:  "is not valid on a container_disk source (a containerDisk is not a PVC-backed volume)",
			})
		}
	}

	// gpus[]: exactly one of resource_name / device_name per entry.
	for i, g := range body.GPUs {
		hasRes := g.ResourceName != ""
		hasDev := g.DeviceName != ""
		if hasRes == hasDev {
			diags.Items = append(diags.Items, spec.Diagnostic{
				Severity: "error",
				Path:     fmt.Sprintf("gpus[%d]", i),
				Message:  "must set exactly one of resource_name (a cluster device-plugin resource) or device_name (a specific host device)",
			})
		}
	}

	// instancetype XOR explicit cpu.
	if body.Instancetype != "" && (body.CPU.Cores != 0 || body.CPU.Sockets != 0 || body.CPU.Threads != 0) {
		diags.Items = append(diags.Items, spec.Diagnostic{
			Severity: "error",
			Path:     "instancetype",
			Message:  "is mutually exclusive with an explicit cpu block (the instancetype owns the domain CPU)",
		})
	}
	return diags, nil
}
