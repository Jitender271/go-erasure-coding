package main

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/klauspost/reedsolomon"
)

func main() {
	// Configuration with validation
	const (
		dataShards   = 4
		parityShards = 2
		totalShards  = dataShards + parityShards
		maxShardSize = 1 << 24 // 16MB max per shard (adjust as needed)
	)

	// Validate configuration before proceeding
	if dataShards <= 0 || parityShards <= 0 {
		log.Fatal("Both dataShards and parityShards must be positive integers")
	}

	// 1. Create a Reed-Solomon encoder with validation
	enc, err := reedsolomon.New(dataShards, parityShards)
	if err != nil {
		log.Fatalf("Failed to create encoder: %v", err)
	}

	fmt.Println("Reed-Solomon encoder created successfully")
	fmt.Printf("Configuration: %d data shards + %d parity shards = %d total shards\n",
		dataShards, parityShards, totalShards)
	fmt.Printf("Can recover from up to %d lost shards\n", parityShards)

	// 2. Prepare our original data with size validation
	originalData := []byte("This is our super important data that needs protection. " +
		"It contains critical information that we can't afford to lose!")

	if len(originalData) == 0 {
		log.Fatal("Input data cannot be empty")
	}
	if len(originalData) > maxShardSize*dataShards {
		log.Fatal("Input data too large for current configuration")
	}

	// Pad data to fit shard boundaries if needed
	paddedLength := len(originalData)
	if len(originalData)%dataShards != 0 {
		padding := dataShards - (len(originalData) % dataShards)
		originalData = append(originalData, make([]byte, padding)...)
		fmt.Printf("Added %d bytes of padding to align with shard boundaries\n", padding)
	}

	fmt.Printf("\nOriginal data (%d bytes):\n%s\n", paddedLength, originalData[:paddedLength])

	// 3. Split the data into shards with error handling
	shards, err := enc.Split(originalData)
	if err != nil {
		log.Fatalf("Failed to split data: %v", err)
	}

	fmt.Println("\nData split into shards:")
	printShards(shards, dataShards, parityShards, paddedLength)

	// 4. Encode parity shards with progress indication
	fmt.Print("\nGenerating parity shards... ")
	err = enc.Encode(shards)
	if err != nil {
		log.Fatalf("Failed to encode parity: %v", err)
	}
	fmt.Println("Done")

	fmt.Println("\nAll shards (data + parity):")
	printShards(shards, dataShards, parityShards, paddedLength)

	// 5. Verify initial shard integrity
	fmt.Print("\nVerifying shard integrity... ")
	ok, err := enc.Verify(shards)
	if err != nil {
		log.Fatalf("Verification failed: %v", err)
	}
	if ok {
		fmt.Println("All shards intact and valid")
	} else {
		log.Fatal("Shard verification failed - data corruption detected")
	}

	// 6. Simulate data loss scenarios
	lostShards := []int{0, 4} // Losing data shard 0 and parity shard 1
	fmt.Printf("\nSimulating loss of shards: %v\n", lostShards)

	// Create a copy of shards for simulation (preserve originals)
	corruptedShards := make([][]byte, len(shards))
	for i := range shards {
		if contains(lostShards, i) {
			continue // Leave as nil to simulate loss
		}
		corruptedShards[i] = make([]byte, len(shards[i]))
		copy(corruptedShards[i], shards[i])
	}

	fmt.Println("\nCorrupted shards state:")
	printShards(corruptedShards, dataShards, parityShards, paddedLength)

	// 7. Verify corruption is detected
	fmt.Print("\nVerifying corrupted shards... ")
	ok, err = enc.Verify(corruptedShards)
	if err != nil {
		fmt.Printf("Expected verification failure: %v\n", err)
	} else if !ok {
		fmt.Println("Correctly detected missing shards")
	} else {
		log.Fatal("Verification should have failed but didn't")
	}

	// 8. Reconstruct the lost shards
	fmt.Print("\nReconstructing missing shards... ")
	err = enc.Reconstruct(corruptedShards)
	if err != nil {
		log.Fatalf("Failed to reconstruct: %v", err)
	}
	fmt.Println("Success")

	fmt.Println("\nReconstructed shards state:")
	printShards(corruptedShards, dataShards, parityShards, paddedLength)

	// 9. Verify reconstruction
	fmt.Print("\nVerifying reconstructed shards... ")
	ok, err = enc.Verify(corruptedShards)
	if err != nil {
		log.Fatalf("Verification failed: %v", err)
	}
	if ok {
		fmt.Println("All shards valid - data fully recovered")
	} else {
		log.Fatal("Reconstruction verification failed")
	}

	// 10. Join the shards to recover original data
	fmt.Print("\nReassembling original data... ")
	buf := new(bytes.Buffer)
	err = enc.Join(buf, corruptedShards, len(originalData))
	if err != nil {
		log.Fatalf("Failed to join shards: %v", err)
	}
	recoveredData := buf.Bytes()[:paddedLength] // Remove padding
	fmt.Println("Done")

	fmt.Printf("\nRecovered data (%d bytes):\n%s\n", len(recoveredData), recoveredData)

	// 11. Validate data recovery
	if bytes.Equal(originalData[:paddedLength], recoveredData) {
		fmt.Println("\nSUCCESS: Recovered data matches original exactly!")
	} else {
		log.Fatal("\nFAILURE: Recovered data does NOT match original")
	}

	// Bonus: Save shards to files to demonstrate practical use
	saveShardsToFiles(shards, dataShards, "shard_%d.dat", "shard_%d.parity")
}

// printShards prints the shards with proper formatting and length handling
func printShards(shards [][]byte, dataShards, parityShards int, originalLength int) {
	shardSize := originalLength / dataShards
	if originalLength%dataShards != 0 {
		shardSize++
	}

	for i, shard := range shards {
		if shard == nil {
			if i < dataShards {
				fmt.Printf("  Data shard %d: [LOST/MISSING]\n", i)
			} else {
				fmt.Printf("  Parity shard %d: [LOST/MISSING]\n", i-dataShards)
			}
			continue
		}

		if i < dataShards {
			// Calculate actual data length for this shard
			start := i * shardSize
			end := start + shardSize
			if end > originalLength {
				end = originalLength
			}
			actualLength := end - start

			fmt.Printf("  Data shard %d (%d bytes): %q\n",
				i, actualLength, string(shard[:actualLength]))
		} else {
			fmt.Printf("  Parity shard %d (%d bytes): %x...\n",
				i-dataShards, len(shard), shard[:min(8, len(shard))])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func contains(slice []int, val int) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}

func saveShardsToFiles(shards [][]byte, dataShards int, dataPattern, parityPattern string) {
	fmt.Println("\nSaving shards to files...")
	for i, shard := range shards {
		var filename string
		if i < dataShards {
			filename = fmt.Sprintf(dataPattern, i)
		} else {
			filename = fmt.Sprintf(parityPattern, i-dataShards)
		}

		err := os.WriteFile(filename, shard, 0644)
		if err != nil {
			log.Printf("Failed to save shard %d to %s: %v", i, filename, err)
		} else {
			fmt.Printf("  Saved %s (%d bytes)\n", filename, len(shard))
		}
	}
}
