package substratekind

// resolve.go — candy/plugin-substrate's OpResolve leg (the substrate-template
// de-type, Cutover I): project an authored TEMPLATE of ANY substrate word
// (local/pod/kubernetes/kindcluster/kubevirt/android/vm/firecracker — the
// substrateWords list) into a Resolved* envelope the kernel consumes without
// importing the concrete spec.Local / spec.Android / …
//
// The envelope types are the CONTRACT's (#ResolvedPod et al., spec/schema/
// substrate_template.cue) because the kernel reads them; only the AUTHORED body
// type moved into a plugin's own CUE module (the pod body today, schema/pod.cue),
// and an arm whose body moved decodes the authored bytes straight into the
// contract envelope instead of through a plugin-owned Go type.

import (
	"encoding/json"
	"fmt"

	"github.com/opencharly/spec/spec"
)

func resolveSubstrateTemplate(in spec.SubstrateTemplateResolveRequest) ([]byte, error) {
	switch {
	case in.Local != nil:
		// The local body's authored fields ARE the #ResolvedLocal envelope's fields —
		// candy / install_opts / env / description / plan — so the projection decodes
		// STRAIGHT into the contract's envelope and copies nothing by hand. The former
		// `var l spec.Local` plus field-by-field copy existed only because the authored
		// body had a Go type of its own; opencharly/spec#304 unit 4 moved that body into
		// this plugin's CUE module (schema/local.cue), and the contract's own note on
		// #ResolvedLocal already said the envelope is the kernel's read surface.
		reply := spec.LocalResolveReply{Resolved: &spec.ResolvedLocal{Raw: in.Local.Local}}
		if err := json.Unmarshal(in.Local.Local, reply.Resolved); err != nil {
			return nil, fmt.Errorf("local resolve: decode: %w", err)
		}
		return json.Marshal(reply)
	case in.Pod != nil:
		// The pod body's authored fields ARE the #ResolvedPod envelope's fields — box /
		// sidecar / secret / env_default / plan — so the projection decodes STRAIGHT into
		// the contract's envelope and copies nothing by hand. The former
		// `var p spec.Pod` plus field-by-field copy existed only because the authored body
		// had a Go type of its own; opencharly/spec#304 unit 3 moved that body into this
		// plugin's CUE module (schema/pod.cue), and the contract's own note on #ResolvedPod
		// already said the kernel "never" reads a spec.Pod.
		reply := spec.PodResolveReply{Resolved: &spec.ResolvedPod{Raw: in.Pod.Pod}}
		if err := json.Unmarshal(in.Pod.Pod, reply.Resolved); err != nil {
			return nil, fmt.Errorf("pod resolve: decode: %w", err)
		}
		return json.Marshal(reply)
	case in.Kubernetes != nil:
		// The authored fields ARE #ResolvedKubernetes envelope's (kubeconfig_context) — the
		// authored body moved into this plugin's CUE module (opencharly/spec#304 unit 5), so
		// the projection decodes straight into the contract envelope.
		reply := spec.KubernetesResolveReply{Resolved: &spec.ResolvedKubernetes{Raw: in.Kubernetes.Kubernetes}}
		if err := json.Unmarshal(in.Kubernetes.Kubernetes, reply.Resolved); err != nil {
			return nil, fmt.Errorf("kubernetes resolve: decode: %w", err)
		}
		return json.Marshal(reply)
	case in.Kindcluster != nil:
		reply := spec.KindclusterResolveReply{Resolved: &spec.ResolvedKindcluster{Raw: in.Kindcluster.Kindcluster}}
		if err := json.Unmarshal(in.Kindcluster.Kindcluster, reply.Resolved); err != nil {
			return nil, fmt.Errorf("kindcluster resolve: decode: %w", err)
		}
		return json.Marshal(reply)
	case in.KubeVirt != nil:
		// The authored body lives in this plugin's CUE module now (opencharly/spec#304 unit 6),
		// so the projection decodes straight into the contract envelope instead of through a
		// spec-authored Go type.
		reply := spec.KubeVirtResolveReply{Resolved: &spec.ResolvedKubeVirt{Raw: in.KubeVirt.KubeVirt}}
		if err := json.Unmarshal(in.KubeVirt.KubeVirt, reply.Resolved); err != nil {
			return nil, fmt.Errorf("kubevirt resolve: decode: %w", err)
		}
		return json.Marshal(reply)
	case in.Android != nil:
		var a spec.Android
		if err := json.Unmarshal(in.Android.Android, &a); err != nil {
			return nil, fmt.Errorf("android resolve: decode: %w", err)
		}
		reply := spec.AndroidResolveReply{Resolved: &spec.ResolvedAndroid{
			Serial:        a.Serial,
			Device:        a.Device,
			ApiLevel:      a.ApiLevel,
			GoogleAccount: a.GoogleAccount,
			Plan:          a.Plan,
			Box:           a.Box,
			Adb:           a.Adb,
			Raw:           in.Android.Android,
		}}
		return json.Marshal(reply)
	case in.Vm != nil:
		var v spec.Vm
		if err := json.Unmarshal(in.Vm.Vm, &v); err != nil {
			return nil, fmt.Errorf("vm resolve: decode: %w", err)
		}
		reply := spec.VmResolveReply{Resolved: &spec.ResolvedVm{
			Source:    v.Source,
			DiskSize:  v.DiskSize,
			Ram:       v.Ram,
			Cpus:      v.Cpus,
			Machine:   v.Machine,
			Firmware:  v.Firmware,
			Backend:   v.Backend,
			Autostart: v.Autostart,
			Network:   v.Network,
			SSH:       v.SSH,
			CloudInit: v.CloudInit,
			Libvirt:   v.Libvirt,
			Plan:      v.Plan,
			Snapshots: v.Snapshots,
			Raw:       in.Vm.Vm,
		}}
		return json.Marshal(reply)
	case in.Firecracker != nil:
		var fc spec.Firecracker
		if err := json.Unmarshal(in.Firecracker.Firecracker, &fc); err != nil {
			return nil, fmt.Errorf("firecracker resolve: decode: %w", err)
		}
		// The envelope carries the RAW body and nothing more: a `kind: firecracker` template
		// describes the HOST's capabilities (the VMM, the kernel, the network, containerd, the
		// runner pools) and the microVM shape — it names no hostname, because the HOST is the
		// deploy's own venue (its `host:`, exactly as local/vm deploys carry theirs). Whoever
		// resolves the venue therefore fills ResolvedFirecracker.Host/SSHUser; this leg must not
		// invent one from a field the template does not have.
		reply := spec.FirecrackerResolveReply{Resolved: &spec.ResolvedFirecracker{
			Raw: in.Firecracker.Firecracker,
		}}
		return json.Marshal(reply)
	default:
		return nil, fmt.Errorf("substrate template resolve: no arm set")
	}
}
