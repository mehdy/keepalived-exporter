package utils

import (
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// OpenFileWithRetry used to open a file if it didn't exist and retry until the max waiting time.
func OpenFileWithRetry(fileName string) (*os.File, error) {
	openFile := func() (*os.File, error) {
		return os.Open(fileName)
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 10 * time.Millisecond
	b.MaxElapsedTime = 2 * time.Second
	b.Reset()

	return backoff.RetryWithData(openFile, b)
}
