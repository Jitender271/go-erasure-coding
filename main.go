package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/klauspost/reedsolomon"
)

func main() {
	// Configuration
	const dataShards = 4   // Number of data shards
	const parityShards = 2 // Number of parity shards
	const totalShards = dataShards + parityShards

	// 1. Create a Reed-Solomon encoder
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		log.Fatalf("Failed to create encoder: %v", err)
	}

	fmt.Println("Reed-Solomon encoder created successfully")
	fmt.Printf("Configuration: %d data shards + %d parity shards = %d total shards\n",
		dataShards, parityShards, totalShards)

	// 2. Prepare our original data
	originalData := []byte("This is our super important data that needs protection. " +
		"It contains critical information that we can't afford to lose!")

	// Ensure the data length is appropriate for our shard configuration
	if len(originalData)%dataShards != 0 {
		// Pad the data if needed
		padding := make([]byte, dataShards-(len(originalData)%dataShards))
		originalData = append(originalData, padding...)
	}

	fmt.Printf("\nOriginal data (%d bytes):\n%s\n", len(originalData), originalData)

	// 3. Split the data into shards
	shards, err := enc.Split(originalData)
	if err != nil {
		log.Fatalf("Failed to split data: %v", err)
	}

	fmt.Println("\nData split into shards:")
	printShards(shards, dataShards, parityShards)

	// 4. Encode parity shards
	err = enc.Encode(shards)
	if err != nil {
		log.Fatalf("Failed to encode parity: %v", err)
	}

	fmt.Println("\nParity shards generated:")
	printShards(shards, dataShards, parityShards)

	// 5. Verify the shards are valid
	ok, err := enc.Verify(shards)
	if err != nil {
		log.Fatalf("Verification failed: %v", err)
	}
	if ok {
		fmt.Println("\nShard verification successful - all shards intact")
	} else {
		fmt.Println("\nShard verification failed - some shards corrupted")
	}

	// 6. Simulate data loss - let's corrupt/lose some shards
	lostShards := []int{0, 4} // Losing data shard 0 and parity shard 1
	fmt.Printf("\nSimulating loss of shards: %v\n", lostShards)

	// Create a copy of shards for simulation
	corruptedShards := make([][]byte, len(shards))
	copy(corruptedShards, shards)
	for _, lost := range lostShards {
		// Set to nil to simulate complete loss
		corruptedShards[lost] = nil
	}

	// 7. Verify we now have corrupted data
	ok, err = enc.Verify(corruptedShards)
	if err != nil {
		log.Printf("Verification failed (expected): %v", err)
	}
	if !ok {
		fmt.Println("Verification correctly detects we have missing shards")
	}

	// 8. Reconstruct the lost shards
	err = enc.Reconstruct(corruptedShards)
	if err != nil {
		log.Fatalf("Failed to reconstruct: %v", err)
	}

	fmt.Println("\nMissing shards reconstructed successfully")
	printShards(corruptedShards, dataShards, parityShards)

	// 9. Verify reconstruction was successful
	ok, err = enc.Verify(corruptedShards)
	if err != nil {
		log.Fatalf("Verification failed: %v", err)
	}
	if ok {
		fmt.Println("\nFinal verification successful - all data recovered")
	} else {
		fmt.Println("\nFinal verification failed - data recovery incomplete")
	}

	// 10. Join the shards to recover original data
	buf := new(bytes.Buffer)
	err = enc.Join(buf, corruptedShards, len(originalData))
	if err != nil {
		log.Fatalf("Failed to join shards: %v", err)
	}
	recoveredData := buf.Bytes()

	// Remove padding if we added any
	if len(recoveredData) > len(originalData) {
		recoveredData = recoveredData[:len(originalData)]
	}

	fmt.Printf("\nRecovered data (%d bytes):\n%s\n", len(recoveredData), recoveredData)

	// 11. Verify the recovered data matches original
	if bytes.Equal(originalData, recoveredData) {
		fmt.Println("\nSUCCESS: Recovered data matches original exactly!")
	} else {
		fmt.Println("\nFAILURE: Recovered data does NOT match original")
	}
}

// printShards prints the shards in a readable format
func printShards(shards [][]byte, dataShards, parityShards int) {
	for i, shard := range shards {
		if shard == nil {
			if i < dataShards {
				fmt.Printf("  Data shard %d: [LOST]\n", i)
			} else {
				fmt.Printf("  Parity shard %d: [LOST]\n", i-dataShards)
			}
			continue
		}

		if i < dataShards {
			fmt.Printf("  Data shard %d: %s\n", i, string(shard))
		} else {
			fmt.Printf("  Parity shard %d: %x...\n", i-dataShards, shard[:min(16, len(shard))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
