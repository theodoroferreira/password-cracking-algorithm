package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	PasswordLength = 8
	SearchSpace    = 100_000_000
	NumTestRuns    = 20
	DataFileName   = "performance_data.csv"
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

func crackSingleThread(password string) (bool, time.Duration) {
	start := time.Now()
	for i := 0; i < SearchSpace; i++ {
		guess := fmt.Sprintf("%0*d", PasswordLength, i)
		if guess == password {
			return true, time.Since(start)
		}
	}
	return false, time.Since(start)
}

func crackMultiThread(password string, numCores int) (bool, time.Duration) {
	start := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	foundChan := make(chan bool, 1)
	chunkSize := SearchSpace / numCores

	for i := 0; i < numCores; i++ {
		wg.Add(1)
		startRange := i * chunkSize
		endRange := (i + 1) * chunkSize
		if i == numCores-1 {
			endRange = SearchSpace
		}
		go func(start, end int) {
			defer wg.Done()
			worker(ctx, password, start, end, foundChan, cancel)
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

func worker(ctx context.Context, password string, start, end int, foundChan chan<- bool, cancel context.CancelFunc) {
	for i := start; i < end; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			guess := fmt.Sprintf("%0*d", PasswordLength, i)
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

func runSingleThreadTests(passwordsToTest []string) []TestResult {
	var results []TestResult
	fmt.Println("\n--- Starting SINGLE-THREAD Performance Tests ---")

	for runID := 1; runID <= NumTestRuns; runID++ {
		fmt.Printf("\n--- Single-Thread Run %d/%d ---\n", runID, NumTestRuns)
		for _, pwd := range passwordsToTest {
			fmt.Printf("Testing password: %s\n", pwd)
			runtime.GC()
			var startMemStats runtime.MemStats
			runtime.ReadMemStats(&startMemStats)

			_, durationSingle := crackSingleThread(pwd)

			var endMemStats runtime.MemStats
			runtime.ReadMemStats(&endMemStats)

			memAllocSingle := float64(endMemStats.TotalAlloc-startMemStats.TotalAlloc) / (1024 * 1024)
			guesses, _ := strconv.Atoi(pwd)
			gpsSingle := float64(guesses) / durationSingle.Seconds()

			results = append(results, TestResult{
				RunID:            runID,
				AlgorithmType:    "Single-Thread",
				Password:         pwd,
				NumCores:         1,
				TimeToCrackSec:   durationSingle.Seconds(),
				GuessesPerSecond: gpsSingle,
				MemAllocMB:       memAllocSingle,
			})
			fmt.Printf("  -> Found in: %.4f seconds\n", durationSingle.Seconds())
			fmt.Printf("  -> Guesses per second: %.4f\n", gpsSingle)
		}
	}
	return results
}

func runMultiThreadTests(passwordsToTest []string) []TestResult {
	var results []TestResult
	numCores := runtime.NumCPU()
	fmt.Printf("\n--- Starting MULTI-THREAD (%d cores) Performance Tests ---\n", numCores)

	for runID := 1; runID <= NumTestRuns; runID++ {
		fmt.Printf("\n--- Multi-Thread Run %d/%d ---\n", runID, NumTestRuns)
		for _, pwd := range passwordsToTest {
			fmt.Printf("Testing password: %s\n", pwd)
			runtime.GC()
			var startMemStats runtime.MemStats
			runtime.ReadMemStats(&startMemStats)

			_, durationMulti := crackMultiThread(pwd, numCores)

			var endMemStats runtime.MemStats
			runtime.ReadMemStats(&endMemStats)
			memAllocMulti := float64(endMemStats.TotalAlloc-startMemStats.TotalAlloc) / (1024 * 1024)
			guesses, _ := strconv.Atoi(pwd)
			gpsMulti := float64(guesses) / durationMulti.Seconds()

			results = append(results, TestResult{
				RunID:            runID,
				AlgorithmType:    "Multi-Thread",
				Password:         pwd,
				NumCores:         numCores,
				TimeToCrackSec:   durationMulti.Seconds(),
				GuessesPerSecond: gpsMulti,
				MemAllocMB:       memAllocMulti,
			})
			fmt.Printf("  -> Found in: %.4f seconds\n", durationMulti.Seconds())
			fmt.Printf("  -> Guesses per second: %.4f\n", gpsMulti)
		}
	}
	return results
}

func main() {
	var passwordsToTest []string

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter an 8-digit password to crack (or press Enter to use default test set): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if _, err := strconv.Atoi(input); err == nil && len(input) == PasswordLength {
		passwordsToTest = []string{input}
		fmt.Printf("Using user-provided password: %s\n", input)
	} else {
		passwordsToTest = []string{"00100000", "50000000", "99900000"}
		fmt.Println("Invalid or no input. Using default password set for benchmarking.")
	}

	fmt.Println("Starting password cracking performance comparison...")

	singleThreadResults := runSingleThreadTests(passwordsToTest)
	multiThreadResults := runMultiThreadTests(passwordsToTest)

	allResults := append(singleThreadResults, multiThreadResults...)

	err := saveResultsToCSV(allResults)
	if err != nil {
		log.Fatalf("Failed to save results to CSV: %v", err)
	}

	fmt.Printf("\n✅ All tests complete. Performance data saved to %s\n", DataFileName)
}

func saveResultsToCSV(results []TestResult) error {
	file, err := os.Create(DataFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"RunID", "AlgorithmType", "Password", "NumCores", "TimeToCrackSec", "GuessesPerSecond", "MemAllocMB"}
	if err := writer.Write(header); err != nil {
		return err
	}

	for _, r := range results {
		row := []string{
			strconv.Itoa(r.RunID),
			r.AlgorithmType,
			r.Password,
			strconv.Itoa(r.NumCores),
			fmt.Sprintf("%.6f", r.TimeToCrackSec),
			fmt.Sprintf("%.2f", r.GuessesPerSecond),
			fmt.Sprintf("%.6f", r.MemAllocMB),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}
	return nil
}
