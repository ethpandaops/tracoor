package promotion

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/OffchainLabs/go-bitfield"
	"github.com/attestantio/go-eth2-client/spec/altair"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/creasty/defaults"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/ethpandaops/tracoor/pkg/compression"
	"github.com/ethpandaops/tracoor/pkg/server/persistence"
	"github.com/ethpandaops/tracoor/pkg/store"
)

const (
	testNetwork = "testnet"
	testNode    = "cl-1-test"
	testMeta    = "test"
)

func fillRoot(b byte) phase0.Root {
	var r phase0.Root
	for i := range r {
		r[i] = b
	}

	return r
}

// slotRoot derives a deterministic, kind-tagged root for a slot.
func slotRoot(kind byte, slot uint64) phase0.Root {
	var r phase0.Root

	r[0] = kind
	binary.BigEndian.PutUint64(r[1:9], slot)

	return r
}

func blockRootFor(slot uint64) phase0.Root { return slotRoot(0xbb, slot) }
func stateRootFor(slot uint64) phase0.Root { return slotRoot(0xaa, slot) }

// testBlock builds a valid altair SignedBeaconBlock and returns its raw SSZ.
func testBlock(t *testing.T, slot uint64, parentRoot, stateRoot phase0.Root, mutate func(body *altair.BeaconBlockBody)) []byte {
	t.Helper()

	bits := bitfield.NewBitvector512()
	for i := range bits.Len() {
		bits.SetBitAt(i, true)
	}

	block := &altair.SignedBeaconBlock{
		Message: &altair.BeaconBlock{
			Slot:          phase0.Slot(slot),
			ProposerIndex: 1,
			ParentRoot:    parentRoot,
			StateRoot:     stateRoot,
			Body: &altair.BeaconBlockBody{
				ETH1Data:      &phase0.ETH1Data{BlockHash: make([]byte, 32)},
				SyncAggregate: &altair.SyncAggregate{SyncCommitteeBits: bits},
			},
		},
	}

	if mutate != nil {
		mutate(block.Message.Body)
	}

	raw, err := block.MarshalSSZ()
	require.NoError(t, err)

	return raw
}

func withAttesterSlashing(body *altair.BeaconBlockBody) {
	data := &phase0.AttestationData{
		BeaconBlockRoot: fillRoot(0x01),
		Source:          &phase0.Checkpoint{Root: fillRoot(0x02)},
		Target:          &phase0.Checkpoint{Root: fillRoot(0x03)},
	}
	attestation := &phase0.IndexedAttestation{AttestingIndices: []uint64{1}, Data: data}
	body.AttesterSlashings = []*phase0.AttesterSlashing{{Attestation1: attestation, Attestation2: attestation}}
}

func withSparseSyncBits(body *altair.BeaconBlockBody) {
	bits := bitfield.NewBitvector512()
	for i := range uint64(100) {
		bits.SetBitAt(i, true)
	}

	body.SyncAggregate.SyncCommitteeBits = bits
}

// testState builds bytes that satisfy the BeaconState fixed prefix; the
// promotion service never decodes states, so the tail is filler.
func testState(gvr phase0.Root, slot uint64) []byte {
	buf := make([]byte, 128)
	binary.LittleEndian.PutUint64(buf[0:8], 1700000000)
	copy(buf[8:40], gvr[:])
	binary.LittleEndian.PutUint64(buf[40:48], slot)

	for i := 48; i < len(buf); i++ {
		buf[i] = 0x42
	}

	return buf
}

// encoded compresses raw the way the capture pipeline would have under the
// given algorithm, so seeded buffer objects match their recorded
// Content-Encoding.
//
// Gzip is written with the standard library on purpose: the compressor is
// decode-only for it, and these fixtures stand in for objects captured before
// the move to zstd.
func encoded(t *testing.T, raw []byte, algorithm *compression.CompressionAlgorithm) []byte {
	t.Helper()

	switch algorithm {
	case compression.None:
		return raw
	case compression.Gzip:
		var buf bytes.Buffer

		writer := gzip.NewWriter(&buf)
		_, err := writer.Write(raw)
		require.NoError(t, err)
		require.NoError(t, writer.Close())

		return buf.Bytes()
	default:
		out, err := compression.NewCompressor().Compress(&raw, algorithm)
		require.NoError(t, err)

		return out
	}
}

type env struct {
	t         *testing.T
	conf      *Config
	db        *persistence.Indexer
	buffer    store.Store
	corpusDir string
	promoter  *Promoter

	// encoding is what the capture pipeline wrote the seeded buffer objects
	// with. It defaults to the pipeline's current default so the suite
	// exercises the real path; a test may set it to a legacy algorithm to
	// prove older objects still promote.
	encoding *compression.CompressionAlgorithm
}

func newEnv(t *testing.T, mutate func(*Config)) *env {
	t.Helper()

	ctx := t.Context()

	db, _, err := persistence.NewMockIndexer()
	require.NoError(t, err)

	buffer, err := store.NewFSStore("promotion_test", logrus.New(), &store.FSStoreConfig{BasePath: t.TempDir()}, store.DefaultOptions().SetMetricsEnabled(false))
	require.NoError(t, err)

	corpusDir := t.TempDir()
	conf := testConfig(t, corpusDir)

	if mutate != nil {
		mutate(conf)
	}

	promoter, err := NewPromoter(ctx, logrus.New(), conf, db, buffer)
	require.NoError(t, err)

	return &env{t: t, conf: conf, db: db, buffer: buffer, corpusDir: corpusDir, promoter: promoter, encoding: compression.Default}
}

// restart replaces the promoter with a fresh instance sharing the same DB and
// stores, simulating a service restart with no persistent cursor.
func (e *env) restart() {
	e.t.Helper()

	promoter, err := NewPromoter(e.t.Context(), logrus.New(), e.conf, e.db, e.buffer)
	require.NoError(e.t, err)

	e.promoter = promoter
}

func testConfig(t *testing.T, corpusDir string) *Config {
	t.Helper()

	doc := fmt.Sprintf(`
enabled: true
lagEpochs: 2
checkInterval: 30s
slotsPerEpoch: 32
secondsPerSlot: 2s
rateCapPerHour: 1000
rareCapPerHour: 120
baselineEveryNEpochs: 8
gapTrigger: 2
syncParticipationFloor: 0.95
store:
  type: fs
  config:
    base_path: %s
`, corpusDir)

	// Defaults first, then the document - exactly the order the server uses,
	// so a field the document omits behaves in tests as it will in prod.
	conf := &Config{}
	require.NoError(t, defaults.Set(conf))
	require.NoError(t, yaml.Unmarshal([]byte(doc), conf))

	return conf
}

// storeConfig builds a store.Config the way yaml loading would, so the raw
// message behind it is a real unmarshaller rather than a zero value.
func storeConfig(t *testing.T, storeType string, config map[string]string) store.Config {
	t.Helper()

	doc := fmt.Sprintf("type: %s\nconfig:\n", storeType)
	for key, value := range config {
		doc += fmt.Sprintf("  %s: %q\n", key, value)
	}

	conf := store.Config{}
	require.NoError(t, yaml.Unmarshal([]byte(doc), &conf))

	return conf
}

func (e *env) insertBlockRow(node string, slot, epoch uint64, rootHex, location string, fetchedAt time.Time) {
	e.t.Helper()

	require.NoError(e.t, e.db.InsertBeaconBlock(e.t.Context(), &persistence.BeaconBlock{
		ID:                   uuid.New().String(),
		Node:                 node,
		Slot:                 int64(slot),
		Epoch:                int64(epoch),
		BlockRoot:            rootHex,
		FetchedAt:            fetchedAt,
		ContentEncoding:      e.encoding.ContentEncoding,
		Location:             location,
		Network:              testNetwork,
		BeaconImplementation: testMeta,
		NodeVersion:          testMeta,
	}))
}

func (e *env) seedBlock(node string, slot, epoch uint64, root phase0.Root, raw []byte, fetchedAt time.Time) {
	e.t.Helper()

	rootHex := root.String()
	location := fmt.Sprintf("beacon_blocks/%s/%d/%s/%s", testNetwork, slot, node, rootHex)
	compressed := encoded(e.t, raw, e.encoding)

	_, err := e.buffer.SaveBeaconBlock(e.t.Context(), &store.SaveParams{Data: bytes.NewReader(compressed), Location: location, ContentEncoding: e.encoding.ContentEncoding})
	require.NoError(e.t, err)

	e.insertBlockRow(node, slot, epoch, rootHex, location, fetchedAt)
}

func (e *env) seedState(node string, slot uint64, stateRoot phase0.Root, raw []byte, fetchedAt time.Time) {
	e.t.Helper()

	location := fmt.Sprintf("beacon_states/%s/%d/%s/%s", testNetwork, slot, node, stateRoot.String())
	compressed := encoded(e.t, raw, e.encoding)

	_, err := e.buffer.SaveBeaconState(e.t.Context(), &store.SaveParams{Data: bytes.NewReader(compressed), Location: location, ContentEncoding: e.encoding.ContentEncoding})
	require.NoError(e.t, err)

	require.NoError(e.t, e.db.InsertBeaconState(e.t.Context(), &persistence.BeaconState{
		ID:                   uuid.New().String(),
		Node:                 node,
		Slot:                 int64(slot),
		Epoch:                int64(slot / 32),
		StateRoot:            stateRoot.String(),
		FetchedAt:            fetchedAt,
		ContentEncoding:      e.encoding.ContentEncoding,
		Location:             location,
		Network:              testNetwork,
		BeaconImplementation: testMeta,
		NodeVersion:          testMeta,
	}))
}

// seedChain seeds a linear chain of blocks at the given slots (each pointing
// at the previous via parent_root) with a matching post-state per slot.
// Returns the raw block bytes per slot.
func (e *env) seedChain(gvr phase0.Root, slots ...uint64) map[uint64][]byte {
	e.t.Helper()

	raws := make(map[uint64][]byte, len(slots))
	now := time.Now()

	for i, slot := range slots {
		parent := blockRootFor(slot - 1)
		if i > 0 {
			parent = blockRootFor(slots[i-1])
		}

		raw := testBlock(e.t, slot, parent, stateRootFor(slot), nil)
		raws[slot] = raw

		e.seedBlock(testNode, slot, slot/32, blockRootFor(slot), raw, now)
		e.seedState(testNode, slot, stateRootFor(slot), testState(gvr, slot), now)
	}

	return raws
}

// seedHeadAnchor seeds a head block so that target = headEpoch - lagEpochs.
func (e *env) seedHeadAnchor(slot uint64) {
	e.t.Helper()

	raw := testBlock(e.t, slot, blockRootFor(slot-1), stateRootFor(slot), nil)
	e.seedBlock(testNode, slot, slot/32, blockRootFor(slot), raw, time.Now())
}

// corpusFiles returns relative path -> sha256 of every object in the corpus.
func (e *env) corpusFiles() map[string]string {
	e.t.Helper()

	files := map[string]string{}

	err := filepath.WalkDir(e.corpusDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if d.IsDir() {
			return nil
		}

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}

		rel, relErr := filepath.Rel(e.corpusDir, path)
		if relErr != nil {
			return relErr
		}

		files[filepath.ToSlash(rel)] = sha256Hex(data)

		return nil
	})
	require.NoError(e.t, err)

	return files
}

func (e *env) findManifests() []string {
	e.t.Helper()

	manifests := []string{}

	for path := range e.corpusFiles() {
		if strings.HasSuffix(path, captureManifestName) {
			manifests = append(manifests, path)
		}
	}

	return manifests
}

func (e *env) readManifest(path string) *Manifest {
	e.t.Helper()

	data, err := os.ReadFile(filepath.Join(e.corpusDir, filepath.FromSlash(path)))
	require.NoError(e.t, err)

	manifest := &Manifest{}
	require.NoError(e.t, json.Unmarshal(data, manifest))

	return manifest
}

// purge removes every indexed block and state for the test network, standing
// in for the retention reaper.
func (e *env) purge() {
	e.t.Helper()

	blocks, err := e.db.ListBeaconBlock(e.t.Context(), &persistence.BeaconBlockFilter{Network: ptr(testNetwork)}, &persistence.PaginationCursor{Limit: 10000, OrderBy: "slot ASC"})
	require.NoError(e.t, err)

	for _, row := range blocks {
		require.NoError(e.t, e.db.RemoveBeaconBlock(e.t.Context(), row.ID))
	}

	states, err := e.db.ListBeaconState(e.t.Context(), &persistence.BeaconStateFilter{Network: ptr(testNetwork)}, &persistence.PaginationCursor{Limit: 10000, OrderBy: "slot ASC"})
	require.NoError(e.t, err)

	for _, row := range states {
		require.NoError(e.t, e.db.RemoveBeaconState(e.t.Context(), row.ID))
	}
}

// expireIdentity ages out the cached genesis validators root so the next tick
// re-resolves it.
func (e *env) expireIdentity() {
	e.t.Helper()

	e.promoter.network(testNetwork).gvrResolved = time.Now().Add(-2 * networkIdentityTTL)
}

func ptr[T any](v T) *T { return &v }

// requireWriteProtectable skips a test that relies on a read-only directory
// actually blocking writes. Root ignores the mode bits, and the test would
// then assert nothing at all.
func requireWriteProtectable(t *testing.T) {
	t.Helper()

	if os.Geteuid() == 0 {
		t.Skip("running as root: read-only directories do not block writes")
	}
}

func (e *env) readCorpusObject(path string) []byte {
	e.t.Helper()

	data, err := os.ReadFile(filepath.Join(e.corpusDir, filepath.FromSlash(path)))
	require.NoError(e.t, err)

	return data
}
