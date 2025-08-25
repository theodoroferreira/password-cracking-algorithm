package main

import (
	"fmt"
	"time"
)

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
