package utils

import (
	"os"
	"time"
)

// OpenFileWithRetry used to open a file if it didn't exist and retry until the max waiting time
func OpenFileWithRetry(fileName string, firstTryTime, maxWaitTime time.Duration) (*os.File, error) {
	waitTime := firstTryTime
	startTime := time.Now()

	for {
		file, err := os.Open(fileName)
		if err == nil {
			return file, nil
		}

		if os.IsNotExist(err) {
			time.Sleep(waitTime)

			waitTime = waitTime * 2
			if waitTime >= maxWaitTime/2 {
				waitTime = maxWaitTime / 2
			}
		} else {
			return nil, err
		}

		if time.Since(startTime) >= maxWaitTime {
			return nil, os.ErrNotExist
		}
	}
}
