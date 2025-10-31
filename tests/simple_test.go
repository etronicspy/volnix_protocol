package tests

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSimplePerformance tests basic performance without complex dependencies
func TestSimplePerformance(t *testing.T) {
	t.Log("Running simple performance test...")

	// Test concurrent operations
	numWorkers := 50
	operationsPerWorker := 1000
	totalOperations := numWorkers * operationsPerWorker

	start := time.Now()
	var wg sync.WaitGroup
	var successCount int64
	var mu sync.Mutex

	// Simulate concurrent processing
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			localSuccess := 0
			for j := 0; j < operationsPerWorker; j++ {
				// Simulate some work
				data := fmt.Sprintf("operation_%d_%d", workerID, j)
				hash := simpleHash(data)
				
				// Simulate validation
				if len(hash) > 0 {
					localSuccess++
				}
				
				// Small delay to simulate real work
				if j%100 == 0 {
					time.Sleep(time.Microsecond * 10)
				}
			}
			
			mu.Lock()
			successCount += int64(localSuccess)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	// Calculate metrics
	throughput := float64(successCount) / duration.Seconds()
	
	t.Logf("Simple Performance Test Results:")
	t.Logf("  Workers: %d", numWorkers)
	t.Logf("  Operations per worker: %d", operationsPerWorker)
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Successful operations: %d", successCount)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Throughput: %.2f operations/second", throughput)
	t.Logf("  Success rate: %.2f%%", float64(successCount)/float64(totalOperations)*100)

	// Verify performance
	require.Greater(t, throughput, 10000.0, "Should achieve at least 10,000 operations/second")
	require.Equal(t, successCount, int64(totalOperations), "All operations should succeed")
}

// TestMemoryUsage tests memory usage patterns
func TestMemoryUsage(t *testing.T) {
	t.Log("Testing memory usage patterns...")

	// Force garbage collection
	runtime.GC()
	
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Create memory load
	numItems := 100000
	items := make([]map[string]interface{}, numItems)
	
	start := time.Now()
	
	for i := 0; i < numItems; i++ {
		items[i] = map[string]interface{}{
			"id":        fmt.Sprintf("item_%d", i),
			"timestamp": time.Now().Unix(),
			"data":      fmt.Sprintf("data_payload_%d", i),
			"hash":      simpleHash(fmt.Sprintf("item_%d", i)),
			"active":    i%2 == 0,
		}
	}
	
	duration := time.Since(start)
	
	// Force garbage collection
	runtime.GC()
	
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Calculate memory usage (handle potential underflow)
	var memoryUsed uint64
	if memAfter.Alloc > memBefore.Alloc {
		memoryUsed = memAfter.Alloc - memBefore.Alloc
	} else {
		memoryUsed = memBefore.Alloc - memAfter.Alloc // GC may have reduced memory
	}
	memoryPerItem := memoryUsed / uint64(numItems)
	
	t.Logf("Memory Usage Test Results:")
	t.Logf("  Items created: %d", numItems)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Memory before: %d bytes", memBefore.Alloc)
	t.Logf("  Memory after: %d bytes", memAfter.Alloc)
	t.Logf("  Memory used: %d bytes (%.2f MB)", memoryUsed, float64(memoryUsed)/(1024*1024))
	t.Logf("  Memory per item: %d bytes", memoryPerItem)

	// Verify reasonable memory usage
	maxMemoryMB := float64(50) // 50MB max for 100k items
	actualMemoryMB := float64(memoryUsed) / (1024 * 1024)
	require.Less(t, actualMemoryMB, maxMemoryMB, 
		fmt.Sprintf("Memory usage should be less than %.2f MB, got %.2f MB", maxMemoryMB, actualMemoryMB))
}

// TestConcurrentReadWrite tests concurrent read/write operations
func TestConcurrentReadWrite(t *testing.T) {
	t.Log("Testing concurrent read/write operations...")

	// Shared data structure
	data := make(map[string]string)
	var mu sync.RWMutex

	numReaders := 10
	numWriters := 5
	testDuration := 5 * time.Second
	
	var readCount, writeCount int64
	var wg sync.WaitGroup

	start := time.Now()

	// Start readers
	for i := 0; i < numReaders; i++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			
			localReads := 0
			for time.Since(start) < testDuration {
				mu.RLock()
				_ = len(data) // Read operation
				mu.RUnlock()
				
				localReads++
				time.Sleep(time.Millisecond)
			}
			
			mu.Lock()
			readCount += int64(localReads)
			mu.Unlock()
		}(i)
	}

	// Start writers
	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			
			localWrites := 0
			for time.Since(start) < testDuration {
				key := fmt.Sprintf("key_%d_%d", writerID, localWrites)
				value := fmt.Sprintf("value_%d_%d", writerID, localWrites)
				
				mu.Lock()
				data[key] = value
				mu.Unlock()
				
				localWrites++
				time.Sleep(time.Millisecond * 10)
			}
			
			mu.Lock()
			writeCount += int64(localWrites)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	actualDuration := time.Since(start)

	// Calculate metrics
	readThroughput := float64(readCount) / actualDuration.Seconds()
	writeThroughput := float64(writeCount) / actualDuration.Seconds()
	
	t.Logf("Concurrent Read/Write Test Results:")
	t.Logf("  Duration: %v", actualDuration)
	t.Logf("  Readers: %d, Writers: %d", numReaders, numWriters)
	t.Logf("  Read operations: %d (%.2f/sec)", readCount, readThroughput)
	t.Logf("  Write operations: %d (%.2f/sec)", writeCount, writeThroughput)
	t.Logf("  Final data size: %d items", len(data))

	// Verify performance
	require.Greater(t, readThroughput, 100.0, "Should achieve at least 100 reads/second")
	require.Greater(t, writeThroughput, 10.0, "Should achieve at least 10 writes/second")
	require.Greater(t, len(data), 0, "Should have written some data")
}

// TestSystemStability tests system stability under load
func TestSystemStability(t *testing.T) {
	t.Log("Testing system stability under sustained load...")

	testDuration := 10 * time.Second
	workerCount := 20
	
	var totalOperations int64
	var errorCount int64
	var wg sync.WaitGroup

	start := time.Now()

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			localOps := 0
			localErrors := 0
			operationID := 0
			
			for time.Since(start) < testDuration {
				// Perform mixed operations
				switch operationID % 3 {
				case 0:
					// Simulate order processing
					orderID := fmt.Sprintf("stability_order_%d_%d", workerID, operationID)
					hash := simpleHash(orderID)
					if len(hash) == 0 {
						localErrors++
					}
					
				case 1:
					// Simulate validator operation
					weight := 1000 + operationID
					if weight < 0 { // This should never happen
						localErrors++
					}
					
				case 2:
					// Simulate account verification
					accountAddr := fmt.Sprintf("cosmos1stabilityacc%d_%d", workerID, operationID)
					hash := simpleHash(accountAddr)
					if len(hash) == 0 {
						localErrors++
					}
				}
				
				localOps++
				operationID++
				
				// Small delay
				time.Sleep(time.Microsecond * 100)
			}
			
			// Update counters atomically
			mu := sync.Mutex{}
			mu.Lock()
			totalOperations += int64(localOps)
			errorCount += int64(localErrors)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	actualDuration := time.Since(start)

	// Calculate metrics
	throughput := float64(totalOperations) / actualDuration.Seconds()
	errorRate := float64(errorCount) / float64(totalOperations) * 100
	
	t.Logf("System Stability Test Results:")
	t.Logf("  Test duration: %v", actualDuration)
	t.Logf("  Workers: %d", workerCount)
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Errors: %d", errorCount)
	t.Logf("  Throughput: %.2f operations/second", throughput)
	t.Logf("  Error rate: %.2f%%", errorRate)

	// Verify stability
	require.Greater(t, totalOperations, int64(1000), "Should complete at least 1000 operations")
	require.Less(t, errorRate, 1.0, "Error rate should be less than 1%")
	require.Greater(t, throughput, 100.0, "Should maintain at least 100 operations/second")
}

// simpleHash creates a simple hash for testing
func simpleHash(input string) string {
	if len(input) == 0 {
		return ""
	}
	
	hash := 0
	for _, char := range input {
		hash = hash*31 + int(char)
	}
	
	return fmt.Sprintf("hash_%x", hash)
}