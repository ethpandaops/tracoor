package promotion

import "time"

// manifestSchema is the manifest schema version.
const manifestSchema = 1

// Branch labels for the promotion.branch manifest field.
const (
	BranchCanonical = "canonical"
	BranchOrphaned  = "orphaned"
)

// Manifest is the claims-as-of-promotion-time record written next to every
// promoted capture. Corrections are additive or omitted, never in-place: a
// wrong label poisons downstream consumers, a missing one is honest.
//
//nolint:tagliatelle // the manifest schema is snake_case by design; it is a consumer-facing contract.
type Manifest struct {
	Schema     int               `json:"schema"`
	Tool       string            `json:"tool"`
	PromotedAt time.Time         `json:"promoted_at"`
	Network    ManifestNetwork   `json:"network"`
	Fork       ManifestFork      `json:"fork"`
	Block      ManifestBlock     `json:"block"`
	Prestate   *ManifestPrestate `json:"prestate,omitempty"`
	Promotion  ManifestPromotion `json:"promotion"`
	Context    ManifestContext   `json:"context"`
}

//nolint:tagliatelle // see Manifest.
type ManifestNetwork struct {
	// Name qualifies the config name with the genesis validators root:
	// devnets are torn down and recreated under the same name constantly,
	// and the GVR prefix is the only stable, cheap disambiguator.
	Name                  string `json:"name"`
	ConfigName            string `json:"config_name"`
	GenesisValidatorsRoot string `json:"genesis_validators_root"`
	SlotsPerEpoch         uint64 `json:"slots_per_epoch"`
	SecondsPerSlot        uint64 `json:"seconds_per_slot"`
}

type ManifestFork struct {
	// Name is the decoded fork name, or "unknown" when no supported fork
	// decodes the block.
	Name string `json:"name"`
}

//nolint:tagliatelle // see Manifest.
type ManifestBlock struct {
	Slot       uint64 `json:"slot"`
	Root       string `json:"root"`
	ParentRoot string `json:"parent_root"`
	StateRoot  string `json:"state_root"`
	// SHA256 is computed over exactly the raw (decompressed) bytes written.
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
}

//nolint:tagliatelle // see Manifest.
type ManifestPrestate struct {
	Slot uint64 `json:"slot"`
	// SHA256 doubles as the state's object name under v1/states/.
	SHA256 string `json:"sha256"`
	Size   int    `json:"size"`
	// ExpectedStateRoot is the PARENT block's own state_root field, read
	// from the parent's bytes - keyed by parent_root, therefore
	// branch-exact. Any consumer with an SSZ library can verify
	// hash_tree_root(state) == expected_state_root forever, without
	// tracoor. Omitted rather than substituted when the parent block is
	// missing from the buffer.
	ExpectedStateRoot string `json:"expected_state_root,omitempty"`
	SourceNode        string `json:"source_node"`
}

//nolint:tagliatelle // see Manifest.
type ManifestPromotion struct {
	Triggers []string `json:"triggers"`
	Decoded  bool     `json:"decoded"`
	// PairVerified is false when the pairing claim is unproven (missing
	// data, undecodable fork); both artifacts are still individually real.
	PairVerified bool   `json:"pair_verified"`
	Branch       string `json:"branch"`
	LagEpochs    uint64 `json:"lag_epochs"`
}

// ManifestContext records network-wide conditions as context, never as
// per-slot triggers. The server side has no beacon connection, so unlike the
// prototype it records only what the index can prove; finality fields are
// deliberately absent rather than fabricated.
//
//nolint:tagliatelle // see Manifest.
type ManifestContext struct {
	HeadSlotAtPromotion uint64 `json:"head_slot_at_promotion"`
}
