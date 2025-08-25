package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type TestResult struct {
	RunID            int
	AlgorithmType    string
	Password         string
	NumCores         int
	TimeToCrackSec   float64
	GuessesPerSecond float64
	MemAllocMB       float64
}

func runSingleThreadTests(passwordLength int, searchSpace int64, numTestRuns int, passwordToCrack string) []TestResult {
	var results []TestResult
	fmt.Println("\n--- Starting SINGLE-THREAD Performance Tests ---")

	for runID := 1; runID <= numTestRuns; runID++ {
		fmt.Printf("\n--- Single-Thread Run %d/%d ---\n", runID, numTestRuns)
		fmt.Printf("Testing password: %s\n", passwordToCrack)
		runtime.GC()
		var startMemStats runtime.MemStats
		runtime.ReadMemStats(&startMemStats)

		_, durationSingle := crackSingleThread(passwordToCrack, passwordLength, searchSpace)

		var endMemStats runtime.MemStats
		runtime.ReadMemStats(&endMemStats)
		memAllocSingle := float64(endMemStats.TotalAlloc-startMemStats.TotalAlloc) / (1024 * 1024)
		guesses, _ := strconv.ParseInt(passwordToCrack, 10, 64)
		gpsSingle := float64(guesses) / durationSingle.Seconds()

		results = append(results, TestResult{
			RunID:            runID,
			AlgorithmType:    "Single-Thread",
			Password:         passwordToCrack,
			NumCores:         1,
			TimeToCrackSec:   durationSingle.Seconds(),
			GuessesPerSecond: gpsSingle,
			MemAllocMB:       memAllocSingle,
		})
		fmt.Printf("  -> Found in: %.4f seconds; Guesses per second: %.4f", durationSingle.Seconds(), gpsSingle)
	}
	return results
}

func runMultiThreadTests(passwordLength int, searchSpace int64, numTestRuns int, numCores int, passwordToCrack string) []TestResult {
	var results []TestResult
	fmt.Printf("\n--- Starting MULTI-THREAD (%d cores) Performance Tests ---\n", numCores)

	for runID := 1; runID <= numTestRuns; runID++ {
		fmt.Printf("\n--- Multi-Thread Run %d/%d ---\n", runID, numTestRuns)
		fmt.Printf("Testing password: %s\n", passwordToCrack)
		runtime.GC()
		var startMemStats runtime.MemStats
		runtime.ReadMemStats(&startMemStats)

		_, durationMulti := crackMultiThread(passwordToCrack, passwordLength, searchSpace, numCores)

		var endMemStats runtime.MemStats
		runtime.ReadMemStats(&endMemStats)
		memAllocMulti := float64(endMemStats.TotalAlloc-startMemStats.TotalAlloc) / (1024 * 1024)
		guesses, _ := strconv.ParseInt(passwordToCrack, 10, 64)
		gpsMulti := float64(guesses) / durationMulti.Seconds()

		results = append(results, TestResult{
			RunID:            runID,
			AlgorithmType:    "Multi-Thread",
			Password:         passwordToCrack,
			NumCores:         numCores,
			TimeToCrackSec:   durationMulti.Seconds(),
			GuessesPerSecond: gpsMulti,
			MemAllocMB:       memAllocMulti,
		})
		fmt.Printf("  -> Found in: %.4f seconds; Guesses per second: %.4f\n", durationMulti.Seconds(), gpsMulti)
	}
	return results
}

func main() {
	rand.Seed(time.Now().UnixNano())
	reader := bufio.NewReader(os.Stdin)

	var numTestRuns int
	for {
		fmt.Print("Enter the number of test runs (e.g., 20): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		runs, err := strconv.Atoi(input)
		if err == nil && runs > 0 {
			numTestRuns = runs
			break
		}
		fmt.Println("Invalid input. Please enter a positive number.")
	}

	var passwordLength int
	for {
		fmt.Print("Enter the desired password length (e.g., 8): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		length, err := strconv.Atoi(input)
		if err == nil && length > 0 {
			passwordLength = length
			break
		}
		fmt.Println("Invalid input. Please enter a positive number.")
	}
	searchSpace := int64(math.Pow10(passwordLength))
	fmt.Printf("Password length set to %d. Search space is %d.\n", passwordLength, searchSpace)

	var mode string
	for {
		fmt.Print("\nSelect execution mode:\n 1: Single-Thread Only\n 2: Multi-Thread Only\n 3: Both\nEnter choice (1, 2, or 3): ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "1" || input == "2" || input == "3" {
			mode = input
			break
		}
		fmt.Println("Invalid choice. Please enter 1, 2, or 3.")
	}

	var userNumCores int
	maxCores := runtime.NumCPU()
	if mode == "2" || mode == "3" {
		for {
			fmt.Printf("\nEnter the number of cores to use (1-%d): ", maxCores)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			cores, err := strconv.Atoi(input)
			if err == nil && cores > 0 && cores <= maxCores {
				userNumCores = cores
				break
			}
			fmt.Printf("Invalid input. Please enter a number between 1 and %d.\n", maxCores)
		}
	}

	var passwordToCrack string
	fmt.Printf("\nEnter a specific %d-digit password to test (or press Enter for a random password): ", passwordLength)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if _, err := strconv.Atoi(input); err == nil && len(input) == passwordLength {
		passwordToCrack = input
		fmt.Printf("Using custom password for all runs: %s\n", passwordToCrack)
	} else {
		minPasswordValue := searchSpace / 4
		passwordToCrack = fmt.Sprintf("%0*d", passwordLength, minPasswordValue+rand.Int63n(searchSpace-minPasswordValue))
		fmt.Printf("No valid custom password entered. Using random password for all runs: %s\n", passwordToCrack)
	}

	var awsCfg aws.Config
	var uploadToS3Flag bool
	fmt.Print("\nDo you want to upload the results to AWS S3? (y/n): ")
	input, _ = reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) == "y" {
		uploadToS3Flag = true
		var err error
		awsCfg, err = LoadAWSConfig()
		if err != nil {
			log.Fatalf("Could not load AWS configuration. Have you set your credentials? Error: %v", err)
		}
	}

	fmt.Println("\nStarting password cracking performance comparison...")

	switch mode {
	case "1":
		results := runSingleThreadTests(passwordLength, searchSpace, numTestRuns, passwordToCrack)
		fileName := "performance_data_1_cores.csv"
		handleFileOutput(results, fileName, uploadToS3Flag, awsCfg)
	case "2":
		results := runMultiThreadTests(passwordLength, searchSpace, numTestRuns, userNumCores, passwordToCrack)
		fileName := fmt.Sprintf("performance_data_%d_cores.csv", userNumCores)
		handleFileOutput(results, fileName, uploadToS3Flag, awsCfg)
	case "3":
		singleThreadResults := runSingleThreadTests(passwordLength, searchSpace, numTestRuns, passwordToCrack)
		multiThreadResults := runMultiThreadTests(passwordLength, searchSpace, numTestRuns, userNumCores, passwordToCrack)
		allResults := append(singleThreadResults, multiThreadResults...)
		handleFileOutput(allResults, "performance_data.csv", uploadToS3Flag, awsCfg)
	}

	fmt.Println("\nAll tests complete.")
}

func handleFileOutput(results []TestResult, fileName string, shouldUpload bool, awsCfg aws.Config) {
	err := saveResultsToCSV(results, fileName)
	if err != nil {
		log.Printf("Error saving CSV file: %v", err)
		return
	}
	if shouldUpload {
		err = uploadToS3(fileName, awsCfg)
		if err != nil {
			log.Printf("Error uploading file to S3: %v", err)
		}
	}
}

func saveResultsToCSV(results []TestResult, fileName string) error {
	if len(results) == 0 {
		return nil
	}
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", fileName, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"RunID", "AlgorithmType", "Password", "NumCores", "TimeToCrackSec", "GuessesPerSecond", "MemAllocMB"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header to %s: %w", fileName, err)
	}

	for _, r := range results {
		row := []string{
			strconv.Itoa(r.RunID), r.AlgorithmType, r.Password,
			strconv.Itoa(r.NumCores), fmt.Sprintf("%.6f", r.TimeToCrackSec),
			fmt.Sprintf("%.2f", r.GuessesPerSecond), fmt.Sprintf("%.6f", r.MemAllocMB),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row to %s: %w", fileName, err)
		}
	}
	fmt.Printf("Performance data saved to %s\n", fileName)
	return nil
}
