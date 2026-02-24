package app

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SnapshotManager manages state snapshots for State Sync.
// When snapshotDir is non-empty, snapshots are persisted to disk and survive restarts.
type SnapshotManager struct {
	app         *VolnixApp
	snapshots   map[uint64]*SnapshotInfo
	chunks      map[string][]byte
	mu          sync.RWMutex
	chunkSize   uint32
	snapshotDir string // if set, persist to disk
}

// SnapshotInfo contains information about a snapshot
type SnapshotInfo struct {
	Height      uint64   // Block height of the snapshot
	Format      uint32   // Snapshot format version
	ChunkCount  uint32   // Number of chunks
	Hash        []byte   // Hash of the snapshot
	ChunkHashes []string // Chunk hashes
}

const (
	SnapshotFormatVersion = 1
	DefaultChunkSize      = 1024 * 1024 // 1 MB
)

// NewSnapshotManager creates a new snapshot manager.
// Pass snapshotDir (e.g. home+"/data/snapshots") for disk persistence across restarts.
func NewSnapshotManager(app *VolnixApp) *SnapshotManager {
	sm := &SnapshotManager{
		app:       app,
		snapshots: make(map[uint64]*SnapshotInfo),
		chunks:    make(map[string][]byte),
		chunkSize: DefaultChunkSize,
	}
	return sm
}

// SetSnapshotDir enables disk persistence. Call after app creation with e.g. filepath.Join(homeDir, "data", "snapshots").
func (sm *SnapshotManager) SetSnapshotDir(dir string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.snapshotDir = dir
	if dir != "" {
		_ = os.MkdirAll(dir, 0755)
		_ = sm.loadFromDisk()
	}
}

func (sm *SnapshotManager) loadFromDisk() error {
	if sm.snapshotDir == "" {
		return nil
	}
	metaPath := filepath.Join(sm.snapshotDir, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var meta struct {
		Snapshots []*SnapshotInfo `json:"snapshots"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	for _, s := range meta.Snapshots {
		sm.snapshots[s.Height] = s
		for _, h := range s.ChunkHashes {
			chunkPath := filepath.Join(sm.snapshotDir, "chunks", h)
			if b, err := os.ReadFile(chunkPath); err == nil {
				sm.chunks[h] = b
			}
		}
	}
	return nil
}

func (sm *SnapshotManager) persistToDisk() error {
	if sm.snapshotDir == "" {
		return nil
	}
	chunksDir := filepath.Join(sm.snapshotDir, "chunks")
	_ = os.MkdirAll(chunksDir, 0755)
	snapshots := make([]*SnapshotInfo, 0, len(sm.snapshots))
	for _, s := range sm.snapshots {
		snapshots = append(snapshots, s)
	}
	meta := struct {
		Snapshots []*SnapshotInfo `json:"snapshots"`
	}{Snapshots: snapshots}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(sm.snapshotDir, "metadata.json"), data, 0644)
}

// CreateSnapshot creates a snapshot of the current application state
func (sm *SnapshotManager) CreateSnapshot(height uint64) (*SnapshotInfo, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if snapshot, exists := sm.snapshots[height]; exists {
		return snapshot, nil
	}

	ctx := sm.app.NewContext(false)
	state, err := sm.exportState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export state: %w", err)
	}

	chunks, chunkHashes, err := sm.splitIntoChunks(state)
	if err != nil {
		return nil, fmt.Errorf("failed to split into chunks: %w", err)
	}

	for i, hash := range chunkHashes {
		sm.chunks[hash] = chunks[i]
		if sm.snapshotDir != "" {
			chunksDir := filepath.Join(sm.snapshotDir, "chunks")
			_ = os.MkdirAll(chunksDir, 0755)
			_ = os.WriteFile(filepath.Join(chunksDir, hash), chunks[i], 0644)
		}
	}

	snapshot := &SnapshotInfo{
		Height:      height,
		Format:      SnapshotFormatVersion,
		ChunkCount:  uint32(len(chunks)),
		ChunkHashes: chunkHashes,
	}

	snapshot.Hash = sm.calculateSnapshotHash(snapshot)
	sm.snapshots[height] = snapshot

	if sm.snapshotDir != "" {
		_ = sm.persistToDisk()
	}

	return snapshot, nil
}

// storeEntry represents a single key-value pair within a store for serialization.
type storeEntry struct {
	Key   []byte `json:"k"`
	Value []byte `json:"v"`
}

// storeSnapshot represents all data for a single KV store.
type storeSnapshot struct {
	Name    string       `json:"name"`
	Entries []storeEntry `json:"entries"`
}

// snapshotPayload is the top-level structure serialized into snapshot bytes.
type snapshotPayload struct {
	Height int64            `json:"height"`
	Stores []storeSnapshot  `json:"stores"`
}

// moduleStoreKeys returns the store keys for all custom modules.
func (sm *SnapshotManager) moduleStoreKeys() map[string]*storetypes.KVStoreKey {
	return map[string]*storetypes.KVStoreKey{
		"ident":      sm.app.keyIdent,
		"lizenz":     sm.app.keyLizenz,
		"anteil":     sm.app.keyAnteil,
		"consensus":  sm.app.keyConsensus,
		"governance": sm.app.keyGovernance,
	}
}

// exportState exports all module KV stores into a JSON-encoded byte slice.
func (sm *SnapshotManager) exportState(ctx sdk.Context) ([]byte, error) {
	payload := snapshotPayload{
		Height: ctx.BlockHeight(),
		Stores: make([]storeSnapshot, 0),
	}

	for name, key := range sm.moduleStoreKeys() {
		store := ctx.KVStore(key)
		iter := store.Iterator(nil, nil)

		snap := storeSnapshot{Name: name, Entries: make([]storeEntry, 0)}
		for ; iter.Valid(); iter.Next() {
			snap.Entries = append(snap.Entries, storeEntry{
				Key:   iter.Key(),
				Value: iter.Value(),
			})
		}
		if err := iter.Close(); err != nil {
			ctx.Logger().Error("failed to close iterator", "store", name, "error", err)
		}

		payload.Stores = append(payload.Stores, snap)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal snapshot payload: %w", err)
	}

	return data, nil
}

// splitIntoChunks splits state data into chunks
func (sm *SnapshotManager) splitIntoChunks(data []byte) ([][]byte, []string, error) {
	var chunks [][]byte
	var chunkHashes []string

	for i := 0; i < len(data); i += int(sm.chunkSize) {
		end := i + int(sm.chunkSize)
		if end > len(data) {
			end = len(data)
		}

		chunk := data[i:end]
		chunks = append(chunks, chunk)

		hash := sha256.Sum256(chunk)
		chunkHash := fmt.Sprintf("%x", hash)
		chunkHashes = append(chunkHashes, chunkHash)
	}

	return chunks, chunkHashes, nil
}

// calculateSnapshotHash calculates the hash of a snapshot
func (sm *SnapshotManager) calculateSnapshotHash(snapshot *SnapshotInfo) []byte {
	data := make([]byte, 8+4+4+len(snapshot.ChunkHashes)*32)
	binary.BigEndian.PutUint64(data[0:8], snapshot.Height)
	binary.BigEndian.PutUint32(data[8:12], snapshot.Format)
	binary.BigEndian.PutUint32(data[12:16], snapshot.ChunkCount)

	offset := 16
	for _, chunkHash := range snapshot.ChunkHashes {
		copy(data[offset:offset+32], []byte(chunkHash))
		offset += 32
	}

	hash := sha256.Sum256(data)
	return hash[:]
}

// GetSnapshot returns a snapshot by height
func (sm *SnapshotManager) GetSnapshot(height uint64) (*SnapshotInfo, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshot, exists := sm.snapshots[height]
	return snapshot, exists
}

// GetChunk returns a chunk by hash
func (sm *SnapshotManager) GetChunk(hash string) ([]byte, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	chunk, exists := sm.chunks[hash]
	return chunk, exists
}

// ListSnapshots returns all available snapshots
func (sm *SnapshotManager) ListSnapshots() []*SnapshotInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	snapshots := make([]*SnapshotInfo, 0, len(sm.snapshots))
	for _, snapshot := range sm.snapshots {
		snapshots = append(snapshots, snapshot)
	}

	return snapshots
}

// ApplyChunk applies a chunk to the application state
func (sm *SnapshotManager) ApplyChunk(chunkIndex uint32, chunk []byte, chunkHash string) error {
	hash := sha256.Sum256(chunk)
	expectedHash := fmt.Sprintf("%x", hash)
	if expectedHash != chunkHash {
		return fmt.Errorf("chunk hash mismatch: expected %s, got %s", expectedHash, chunkHash)
	}

	sm.mu.Lock()
	sm.chunks[chunkHash] = chunk
	sm.mu.Unlock()

	return nil
}

// CompleteSnapshot completes the snapshot application process
func (sm *SnapshotManager) CompleteSnapshot(snapshot *SnapshotInfo) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, chunkHash := range snapshot.ChunkHashes {
		if _, exists := sm.chunks[chunkHash]; !exists {
			return fmt.Errorf("missing chunk: %s", chunkHash)
		}
	}

	chunks := make([][]byte, len(snapshot.ChunkHashes))
	for i, chunkHash := range snapshot.ChunkHashes {
		chunks[i] = sm.chunks[chunkHash]
	}

	ctx := sm.app.NewContext(false)
	return sm.importState(ctx, chunks)
}

// importState reconstructs module stores from snapshot chunks.
func (sm *SnapshotManager) importState(ctx sdk.Context, chunks [][]byte) error {
	var stateData []byte
	for _, chunk := range chunks {
		stateData = append(stateData, chunk...)
	}

	var payload snapshotPayload
	if err := json.Unmarshal(stateData, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal snapshot payload: %w", err)
	}

	storeKeys := sm.moduleStoreKeys()

	for _, snap := range payload.Stores {
		key, ok := storeKeys[snap.Name]
		if !ok {
			ctx.Logger().Warn("Skipping unknown store in snapshot", "store", snap.Name)
			continue
		}

		store := ctx.KVStore(key)

		// Clear existing data before import
		iter := store.Iterator(nil, nil)
		keysToDelete := make([][]byte, 0)
		for ; iter.Valid(); iter.Next() {
			keysToDelete = append(keysToDelete, iter.Key())
		}
		if err := iter.Close(); err != nil {
			ctx.Logger().Error("failed to close iterator during import", "store", snap.Name, "error", err)
		}
		for _, k := range keysToDelete {
			store.Delete(k)
		}

		// Write snapshot entries
		for _, entry := range snap.Entries {
			store.Set(entry.Key, entry.Value)
		}

		ctx.Logger().Info("Store imported from snapshot",
			"store", snap.Name,
			"entries", len(snap.Entries),
		)
	}

	ctx.Logger().Info("State imported from snapshot",
		"height", payload.Height,
		"stores", len(payload.Stores),
		"total_size", len(stateData),
	)

	return nil
}

// GetLatestSnapshot returns the latest snapshot
func (sm *SnapshotManager) GetLatestSnapshot() *SnapshotInfo {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var latest *SnapshotInfo
	var maxHeight uint64

	for _, snapshot := range sm.snapshots {
		if snapshot.Height > maxHeight {
			maxHeight = snapshot.Height
			latest = snapshot
		}
	}

	return latest
}
