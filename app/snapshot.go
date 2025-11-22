package app

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// SnapshotManager manages state snapshots for State Sync
type SnapshotManager struct {
	app        *VolnixApp
	snapshots  map[uint64]*SnapshotInfo
	chunks     map[string][]byte // chunk hash -> chunk data
	mu         sync.RWMutex
	chunkSize  uint32 // Size of each chunk in bytes
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
	// SnapshotFormatVersion is the current snapshot format version
	SnapshotFormatVersion = 1
	
	// DefaultChunkSize is the default size of each chunk (1 MB)
	DefaultChunkSize = 1024 * 1024
)

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(app *VolnixApp) *SnapshotManager {
	return &SnapshotManager{
		app:       app,
		snapshots: make(map[uint64]*SnapshotInfo),
		chunks:    make(map[string][]byte),
		chunkSize: DefaultChunkSize,
	}
}

// CreateSnapshot creates a snapshot of the current application state
func (sm *SnapshotManager) CreateSnapshot(height uint64) (*SnapshotInfo, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Check if snapshot already exists
	if snapshot, exists := sm.snapshots[height]; exists {
		return snapshot, nil
	}
	
	// Export application state
	ctx := sm.app.NewContext(false)
	state, err := sm.exportState(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to export state: %w", err)
	}
	
	// Split state into chunks
	chunks, chunkHashes, err := sm.splitIntoChunks(state)
	if err != nil {
		return nil, fmt.Errorf("failed to split into chunks: %w", err)
	}
	
	// Store chunks
	for i, hash := range chunkHashes {
		sm.chunks[hash] = chunks[i]
	}
	
	// Create snapshot info
	snapshot := &SnapshotInfo{
		Height:      height,
		Format:      SnapshotFormatVersion,
		ChunkCount:  uint32(len(chunks)),
		ChunkHashes: chunkHashes,
	}
	
	// Calculate snapshot hash
	snapshot.Hash = sm.calculateSnapshotHash(snapshot)
	
	// Store snapshot
	sm.snapshots[height] = snapshot
	
	return snapshot, nil
}

// exportState exports the current application state
func (sm *SnapshotManager) exportState(ctx sdk.Context) ([]byte, error) {
	// Export state from all modules
	// This is a simplified version - in production, you'd use module managers
	
	// For now, we'll export a basic state representation
	// In a full implementation, this would iterate through all stores and export their data
	
	// Get current block height
	height := ctx.BlockHeight()
	
	// Serialize state (simplified - in production use proper encoding)
	// For now, return a placeholder with height
	stateBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(stateBytes, uint64(height))
	
	return stateBytes, nil
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
		
		// Calculate chunk hash
		hash := sha256.Sum256(chunk)
		chunkHash := fmt.Sprintf("%x", hash)
		chunkHashes = append(chunkHashes, chunkHash)
	}
	
	return chunks, chunkHashes, nil
}

// calculateSnapshotHash calculates the hash of a snapshot
func (sm *SnapshotManager) calculateSnapshotHash(snapshot *SnapshotInfo) []byte {
	// Combine snapshot metadata
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
	// Verify chunk hash
	hash := sha256.Sum256(chunk)
	expectedHash := fmt.Sprintf("%x", hash)
	if expectedHash != chunkHash {
		return fmt.Errorf("chunk hash mismatch: expected %s, got %s", expectedHash, chunkHash)
	}
	
	// Store chunk for later application
	sm.mu.Lock()
	sm.chunks[chunkHash] = chunk
	sm.mu.Unlock()
	
	return nil
}

// CompleteSnapshot completes the snapshot application process
func (sm *SnapshotManager) CompleteSnapshot(snapshot *SnapshotInfo) error {
	// Verify all chunks are present
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	
	for _, chunkHash := range snapshot.ChunkHashes {
		if _, exists := sm.chunks[chunkHash]; !exists {
			return fmt.Errorf("missing chunk: %s", chunkHash)
		}
	}
	
	// Reconstruct state from chunks
	chunks := make([][]byte, len(snapshot.ChunkHashes))
	for i, chunkHash := range snapshot.ChunkHashes {
		chunks[i] = sm.chunks[chunkHash]
	}
	
	// Apply state to application
	// This is a simplified version - in production, you'd properly deserialize and apply state
	ctx := sm.app.NewContext(false)
	return sm.importState(ctx, chunks)
}

// importState imports state from chunks
func (sm *SnapshotManager) importState(ctx sdk.Context, chunks [][]byte) error {
	// Reconstruct state data from chunks
	var stateData []byte
	for _, chunk := range chunks {
		stateData = append(stateData, chunk...)
	}
	
	// Deserialize and apply state
	// This is a simplified version - in production, you'd properly deserialize and apply to stores
	
	// For now, just log the import
	ctx.Logger().Info("State imported from snapshot", "size", len(stateData))
	
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

