package utils

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

func RetryWithBackoff(attempts int, initialDelay time.Duration, maxDelay time.Duration, fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
			println("[Error::RetryWithBackoff] recover from panic", r)
		}
	}()

	delay := initialDelay
	for i := 0; i < attempts; i++ {
		err := fn()
		if err == nil {
			return nil
		}
		if i < attempts-1 {
			jitter := time.Duration(rand.Int63n(int64(delay) / 2))
			sleep := delay + jitter
			if sleep > maxDelay {
				sleep = maxDelay
			}
			println(fmt.Sprintf("retry %d, wait %v, error: %v\n", i+1, sleep, err))
			time.Sleep(sleep)
			delay = time.Duration(math.Min(float64(delay)*2, float64(maxDelay)))
		}
	}
	return errors.New("retry failed")
}
