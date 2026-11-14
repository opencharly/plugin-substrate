// schema/vm.cue — the `kind: vm` ENTITY body, owned by the plugin that SERVES the kind.
//
// THE KIND BODY LIVES HERE. This plugin's `kind: vm` capability declares the gate def
// (`plugin.go`: `InputDef: "#VmValue"`), so the host resolves it from that DECLARATION and
// reads it out of the schema composed from the base and every served plugin unit.
// opencharly/spec#304 unit 8 moved it here.
//
// WHAT CAME WITH IT: seven defs — `#Vm` itself, `schema.#VmBackend`, `schema.#VmBackendName`, `schema.#VmChecksum`,
// `schema.#VmInstaller`, `schema.#VmKeyInjection` and `schema.#VmCharlyInstall` — plus the `#VmValue` twin at the foot.
//
// WHAT STAYED IN THE CONTRACT, and why the line falls here: the resolve ENVELOPE, `#ResolvedVm`,
// type-cites `#VmSource`, `#VmNetwork`, `#VmSsh`, `#VmCloudInit`, `#VmKrun`, `#VmSnapshot` and
// `#LibvirtDomain` (with its sixty-def domain family), and that envelope is a WIRE type read by
// charly's fallback resolver, plugin-check (through `vmshared.VmSpec`) and plugin-deploy-vm. A
// wire type's citations pin their defs in the contract; the body cites them as `schema.#…`.
//
// THE EIGHT VOCABULARY WORDS are cited as `schema.#Vm<X>Word` — the PUBLIC re-exports
// opencharly/spec#320 added, because a plugin cannot reference a private `_<x>Word` and the
// private tables are shared with the keepers above.
//
// THE MODULE'S SINGLE IMPORT STAYS IN `firecracker.cue` (the file that sorts FIRST);
// spec/schemaconcat.ConcatSchema concatenates a module's files in sorted order, and `vm.cue`
// sorts LAST.


#Vm: {
	source: schema.#VmSource @go(Source,type=VmSource)

	disk_size?: string @go(DiskSize)
	ram?:       string
	cpu?:       int & >=1 @go(Cpus,type=int) // yaml key is singular `cpu` (VmSpec.Cpus yaml:"cpu")
	machine?:   or(_machineWord)
	// firmware is REQUIRED-with-default (not optional): an if-guard can only
	// reference a field that always resolves to a concrete value, and an
	// OPTIONAL field — even one carrying a default — errors with "cannot
	// reference optional field" when absent. Required-with-default materializes
	// "bios" on omission (matching the Go empty→bios behavior) AND stays
	// referenceable by the uefi-secure cross-rule below.
	firmware: *"bios" | or(_vmFirmwareWord)

	backend:   schema.#VmBackend
	autostart: *false | true
	// autostart:true requires an authoring backend word: the guard is `or(_backendWord)`, whose
	// rows are `auto` and `libvirt`. `auto` resolves by detection at apply time, so `libvirt` is
	// the explicit choice; a `krun` guest has no persistent daemon at all (and `qemu` is a
	// PROVIDER word in `schema.#VmBackendName`, not a row of this authoring table).
	if autostart {
		backend: or(_backendWord)
	}

	// krun carries the krun backend's knobs. Optional and ignored by the other
	// backends; the consumer applies its defaults when the resolved backend is
	// krun and the block is absent.
	krun?: schema.#VmKrun @go(Krun,optional=nillable)

	// A krun guest's root is a containerd snapshot, not a libvirt domain: its
	// source must be an OCI-image arm (`container_disk` / `bootc`), and the
	// libvirt/qemu-only fields are rejected. `firmware` and `autostart` are
	// required-with-default in this def, so they are VALUE-pinned to their
	// defaults here: an authored uefi-* firmware or autostart:true conflicts,
	// while the materialized defaults pass. Each rejection is deliberately a
	// FIELD-named conflict (null for the struct-valued fields, a negated enum for
	// machine) rather than a bare `_|_`, whose error names only #Vm.
	if backend == "krun" {
		source: schema.#VmSource & {kind: or(_vmKindWord)}
		libvirt?:   null
		autostart:  false
		firmware:   "bios"
		machine?:   !~"^(q35|virt|i440fx|pc)$"
		cloud_init?: null
	}

	// Secure Boot needs Q35 SMM — i440fx can't supply it.
	// machine stays OPTIONAL (Go allows empty machine with uefi-secure), so the
	// constraint is `machine?: !=…`, not `machine: !=…` (the latter would force
	// machine present and false-reject the common omit-machine uefi-secure case).
	if firmware == "uefi-secure" {
		machine?: !="i440fx"
		// Secure Boot requires SMM: the renderer does NOT
		// auto-enable SMM for uefi-secure (buildDomainFeatures only sets it from an
		// explicit libvirt.features.smm), so the user MUST declare it. The `!`
		// required-field markers force libvirt/features/smm to be EXPLICITLY present
		// (a plain `smm: true` would auto-fill and silently pass) and pinned true.
		libvirt!: features!: smm!: true
	}

	network?:    schema.#VmNetwork    @go(Network,optional=nillable)
	ssh?:        schema.#VmSsh        @go(SSH,type=*VmSsh)
	// The explicit pointer type keeps the generated Go field as *VmCloudInit even
	// though the krun guard above adds the `null` arm (a bare `optional=nillable`
	// degrades to an anonymous struct in that case).
	cloud_init?: schema.#VmCloudInit  @go(CloudInit,type=*VmCloudInit)
	// mcp_provide: a kind:vm entity declares its MCP servers (e.g. the
	// systemd-started charly mcp server) exactly like a box does — the P4
	// substrate-neutral surface.
	mcp_provide?: [...schema.#CandyMCPProvide] @go(MCPProvide)
	libvirt?:    schema.#LibvirtDomain @go(Libvirt,type=*LibvirtDomain)

	plan?: [...schema.#Step]
	snapshot?: [...schema.#VmSnapshot] @go(Snapshots)
}







// #VmValue — the HOST-SIDE value gate def for the `vm` kind (the `#PodValue` analogue). @go(-):
// the plugin keeps no typed Go body for it; the resolve envelope carries the raw bytes.
#VmValue: (#Vm | schema.#DeployValue) @go(-)

// --- vocabulary tables (S3), moved with the body ---------------------------------
// The four sets the vm BODY cites and nothing else did (opencharly/spec#304 unit 8): the
// contract keeps only the tables its own keepers cite. `@vocab` names the owner (§3.1):
// these sets are compiled into THIS plugin's module now.
_machine: {q35: {}, virt: {}, i440fx: {}, pc: {}} @vocab(plugin-substrate)
_machineWord: [for w, _ in _machine {w}]

_vmFirmware: {bios: {}, "uefi-insecure": {}, "uefi-secure": {}} @vocab(plugin-substrate)
_vmFirmwareWord: [for w, _ in _vmFirmware {w}]

_backend: {auto: {}, libvirt: {}} @vocab(plugin-substrate)
_backendWord: [for w, _ in _backend {w}]

_vmKind: {container_disk: {}, bootc: {}} @vocab(plugin-substrate)
_vmKindWord: [for w, _ in _vmKind {w}]
