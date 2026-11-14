// schema/vm.cue — the `kind: vm` ENTITY body, owned by the plugin that SERVES the kind.
//
// THE KIND BODY LIVES HERE. This plugin's `kind: vm` capability declares the gate def
// (`plugin.go`: `InputDef: "#VmValue"`), so the host resolves it from that DECLARATION and
// reads it out of the schema composed from the base and every served plugin unit.
// opencharly/spec#304 unit 8 moved it here.
//
// WHAT CAME WITH IT: seven defs — `#Vm` itself, `#VmBackend`, `#VmBackendName`, `#VmChecksum`,
// `#VmInstaller`, `#VmKeyInjection` and `#VmCharlyInstall` — plus the `#VmValue` twin at the foot.
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
	machine?:   or(schema.#VmMachineWord)
	// firmware is REQUIRED-with-default (not optional): an if-guard can only
	// reference a field that always resolves to a concrete value, and an
	// OPTIONAL field — even one carrying a default — errors with "cannot
	// reference optional field" when absent. Required-with-default materializes
	// "bios" on omission (matching the Go empty→bios behavior) AND stays
	// referenceable by the uefi-secure cross-rule below.
	firmware: *"bios" | or(schema.#VmFirmwareWord)

	backend:   #VmBackend
	autostart: *false | true
	// autostart:true requires an explicit backend word: `auto` is a detection placeholder and a
	// krun guest has no persistent daemon, so `libvirt` or `qemu` are the words that can honour it
	// (the guard is the word list; which word actually persists is the backend's business).
	if autostart {
		backend: or(schema.#VmBackendWord)
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
		source: schema.#VmSource & {kind: or(schema.#VmKindWord)}
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

// #VmBackend — the authored/persisted VM-backend selector: the provider words
// plus "auto", which resolves by detection (today: libvirt, then qemu) and is
// never a provider word. The word list lives HERE so the template (#Vm), the
// persisted deploy state (#VmDeployState) and the VM-backend class all read ONE
// source.
#VmBackend: *"auto" | #VmBackendName

// #VmBackendName — the CLOSED VM-backend PROVIDER vocabulary, with no
// resolution selector. `charly task cue-gen` emits it as spec.VmBackends, so a
// resolver validates `--backend` / `CHARLY_VM_BACKEND` / `vm.backend` against the
// schema's own word list instead of carrying literals. `auto` is NOT a member:
// it is a resolution selector, never a provider word.
#VmBackendName: or(schema.#VmBackendNameWord)

#VmChecksum: {
	type?:  "sha256"
	value?: string & =~"^[0-9a-fA-F]{64}$"
}

// #VmInstaller — the unattended-install ANSWERS for a `source.kind: iso` VM.
//
// Distro-AGNOSTIC on purpose: every field here is a question EVERY installer asks.
// Distro-SPECIFIC extras go in `answer:` (typed-open — a distro's installer vocabulary
// is that distro's business, not vm.cue's), reachable from a #DistroInstaller template
// as {{index .Answers "key"}}. The FORMAT that renders these into files lives on the
// distro (#DistroInstaller), exactly as `init:` renders `service:`: archinstall JSON,
// kickstart, preseed, Subiquity autoinstall and AutoYaST are five real formats, each
// keyed by distro and by nothing else.
//
// password ⊻ password_hash is enforced by the vm kind's OWN OpValidate, not by CUE —
// the same reason validateSourceDistro lives there: the host's value gate is
// closedness-only by design, and a disjunction over two optional strings cannot
// express "exactly one present".
#VmInstaller: {
	username?:  string & =~"^[a-z_][a-z0-9_-]{0,31}$"
	full_name?: string
	email?:     string
	// password is PLAINTEXT. The renderer crypt()s it before it reaches any template,
	// so the plaintext exists only in memory and never lands on the answers volume.
	password?:      string & !=""
	password_hash?: string & =~"^\\$(1|5|6|y|2b)\\$"
	// disk is the target block device the installer wipes. It is the guest's view
	// (/dev/vda), not a host path.
	disk?: string & =~"^/dev/[a-z0-9/]+$"
	// encrypt requests full-disk encryption. NOTE: an encrypted unattended install is
	// not fully unattended — someone still types the LUKS passphrase at first boot —
	// and the passphrase is written in PLAINTEXT onto the answers volume, which makes
	// that volume a secret. Source it from the secrets backend, never a committed
	// literal.
	encrypt?:  bool
	keyboard?: string & !=""
	timezone?: string & !=""
	locale?:   string & !=""
	hostname?: string & !=""
	// ssh_authorized_key seeds the created account's ~/.ssh/authorized_keys. On a
	// distro whose installer honours it this is what makes the guest reachable at all
	// (a stock Omarchy install ships openssh with the service disabled and the port
	// closed), so charly defaults it to the public half of the per-VM generated key.
	ssh_authorized_key?: [...(string & !="")]
	// defer_provisioning installs with NO personal details, leaving the first person
	// to boot the machine to create their own user — the imaging-rig mode. Mutually
	// exclusive with the credential fields; enforced in OpValidate.
	defer_provisioning?: bool
	// answer carries per-distro extras the common fields do not model (a Tailscale
	// auth key, a proxy, a licence). Typed-open by design.
	answer?: {[string]: string}
}

#VmKeyInjection: {
	smbios?:     or(schema.#VmSmbiosWord) @go(SMBIOS)
	cloud_init?: or(schema.#VmCloudInitWord) @go(CloudInit)
}

#VmCharlyInstall: {
	// VmCharlyInstall has ONLY `strategy` (the vm-spec skill's url/checksum are
	// STALE — the Go struct dropped them). auto: scp host binary post-boot;
	// scp: explicit form; skip: user-managed.
	strategy?: *"auto" | or(schema.#VmStrategyWord)
}

// #VmValue — the HOST-SIDE value gate def for the `vm` kind (the `#PodValue` analogue). @go(-):
// the plugin keeps no typed Go body for it; the resolve envelope carries the raw bytes.
#VmValue: (#Vm | schema.#DeployValue) @go(-)
