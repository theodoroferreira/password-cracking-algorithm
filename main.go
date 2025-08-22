package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
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

func crackSingleThread(password string, passwordLength int, searchSpace int64) (bool, time.Duration) {
	start := time.Now()
	for i := int64(0); i < searchSpace; i++ {
		guess := fmt.Sprintf("%0*d", passwordLength, i)
		if guess == password {
			return true, time.Since(start)
		}
	}
	return false, time.Since(start)
}

func crackMultiThread(password string, passwordLength int, searchSpace int64, numCores int) (bool, time.Duration) {
	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	foundChan := make(chan bool, 1)
	chunkSize := searchSpace / int64(numCores)

	for i := 0; i < numCores; i++ {
		wg.Add(1)
		startRange := int64(i) * chunkSize
		endRange := (int64(i) + 1) * chunkSize
		if i == numCores-1 {
			endRange = searchSpace
		}
		go func(start, end int64) {
			defer wg.Done()
			worker(ctx, password, passwordLength, start, end, foundChan, cancel)
		}(startRange, endRange)
	}

	wg.Wait()
	close(foundChan)
	duration := time.Since(start)

	if len(foundChan) > 0 {
		return true, duration
	}
	return false, duration
}

func worker(ctx context.Context, password string, passwordLength int, start, end int64, foundChan chan<- bool, cancel context.CancelFunc) {
	for i := start; i < end; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			guess := fmt.Sprintf("%0*d", passwordLength, i)
			if guess == password {
				select {
				case foundChan <- true:
				default:
				}
				cancel()
				return
			}
		}
	}
}

func runSingleThreadTests(passwordsToTest []string, passwordLength int, searchSpace int64, numTestRuns int) []TestResult {
	var results []TestResult
	fmt.Println("\n--- Starting SINGLE-THREAD Performance Tests ---")

	for runID := 1; runID <= numTestRuns; runID++ {
		fmt.Printf("\n--- Single-Thread Run %d/%d ---\n", runID, numTestRuns)
		for _, pwd := range passwordsToTest {
			fmt.Printf("Testing password: %s\n", pwd)
			runtime.GC()
			var startMemStats runtime.MemStats
			runtime.ReadMemStats(&startMemStats)

			_, durationSingle := crackSingleThread(pwd, passwordLength, searchSpace)

			var endMemStats runtime.MemStats
			runtime.ReadMemStats(&endMemStats)
			memAllocSingle := float64(endMemStats.TotalAlloc-startMemStats.TotalAlloc) / (1024 * 1024)
			guesses, _ := strconv.ParseInt(pwd, 10, 64)
			gpsSingle := float64(guesses) / durationSingle.Seconds()

			results = append(results, TestResult{
				RunID: runID, AlgorithmType: "Single-Thread", Password: pwd,
				NumCores: 1, TimeToCrackSec: durationSingle.Seconds(),
				GuessesPerSecond: gpsSingle, MemAllocMB: memAllocSingle,
			})
			fmt.Printf("  -> Found in: %.4f seconds\n", durationSingle.Seconds())
		}
	}
	return results
}

func runMultiThreadTests(passwordsToTest []string, passwordLength int, searchSpace int64, numTestRuns int, numCores int) []TestResult {
	var results []TestResult
	fmt.Printf("\n--- Starting MULTI-THREAD (%d cores) Performance Tests ---\n", numCores)

	for runID := 1; runID <= numTestRuns; runID++ {
		fmt.Printf("\n--- Multi-Thread Run %d/%d ---\n", runID, numTestRuns)
		for _, pwd := range passwordsToTest {
			fmt.Printf("Testing password: %s\n", pwd)
			runtime.GC()
			var startMemStats runtime.MemStats
			runtime.ReadMemStats(&startMemStats)

			_, durationMulti := crackMultiThread(pwd, passwordLength, searchSpace, numCores)

			var endMemStats runtime.MemStats
			runtime.ReadMemStats(&endMemStats)
			memAllocMulti := float64(endMemStats.TotalAlloc-startMemStats.TotalAlloc) / (1024 * 1024)
			guesses, _ := strconv.ParseInt(pwd, 10, 64)
			gpsMulti := float64(guesses) / durationMulti.Seconds()

			results = append(results, TestResult{
				RunID: runID, AlgorithmType: "Multi-Thread", Password: pwd,
				NumCores: numCores, TimeToCrackSec: durationMulti.Seconds(),
				GuessesPerSecond: gpsMulti, MemAllocMB: memAllocMulti,
			})
			fmt.Printf("  -> Found in: %.4f seconds\n", durationMulti.Seconds())
		}
	}
	return results
}

func main() {
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

	var passwordsToTest []string
	midPoint := fmt.Sprintf("%0*d", passwordLength, searchSpace/2)
	endPoint := fmt.Sprintf("%0*d", passwordLength, searchSpace-1)
	defaultPasswords := []string{midPoint, endPoint}

	fmt.Printf("\nEnter a %d-digit password to crack (or press Enter to use default test set [%s, %s]): ", passwordLength, midPoint, endPoint)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if _, err := strconv.Atoi(input); err == nil && len(input) == passwordLength {
		passwordsToTest = []string{input}
		fmt.Printf("Using user-provided password: %s\n", input)
	} else {
		passwordsToTest = defaultPasswords
		fmt.Println("Invalid or no input. Using default password set for benchmarking.")
	}

	fmt.Println("\nStarting password cracking performance comparison...")

	if mode == "1" || mode == "3" {
		results := runSingleThreadTests(passwordsToTest, passwordLength, searchSpace, numTestRuns)
		saveResultsToCSV(results, 1)
	}
	if mode == "2" || mode == "3" {
		results := runMultiThreadTests(passwordsToTest, passwordLength, searchSpace, numTestRuns, userNumCores)
		saveResultsToCSV(results, userNumCores)
	}

	fmt.Println("\n✅ All tests complete.")
}

func saveResultsToCSV(results []TestResult, numCores int) {
	if len(results) == 0 {
		return
	}
	fileName := fmt.Sprintf("performance_data_%d_cores.csv", numCores)
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Failed to create file %s: %v", fileName, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"RunID", "AlgorithmType", "Password", "NumCores", "TimeToCrackSec", "GuessesPerSecond", "MemAllocMB"}
	if err := writer.Write(header); err != nil {
		log.Fatalf("Failed to write header to %s: %v", fileName, err)
	}

	for _, r := range results {
		row := []string{
			strconv.Itoa(r.RunID), r.AlgorithmType, r.Password,
			strconv.Itoa(r.NumCores), fmt.Sprintf("%.6f", r.TimeToCrackSec),
			fmt.Sprintf("%.2f", r.GuessesPerSecond), fmt.Sprintf("%.6f", r.MemAllocMB),
		}
		if err := writer.Write(row); err != nil {
			log.Fatalf("Failed to write row to %s: %v", fileName, err)
		}
	}
	fmt.Printf("Performance data saved to %s\n", fileName)
}
