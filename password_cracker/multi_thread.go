package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

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

	go func() {
		wg.Wait()
		close(foundChan)
	}()

	found := <-foundChan
	duration := time.Since(start)

	return found, duration
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
