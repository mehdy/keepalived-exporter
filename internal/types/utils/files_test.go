package utils

import (
	"os"
	"testing"
	"time"
)

func TestOpenFileWithRetry(t *testing.T) {
	t.Parallel()

	fileName := "/tmp/keepalived-exporter-test.txt"
	testBody := "keepalived-exporter"

	go func() {
		time.Sleep(100 * time.Millisecond)

		_ = os.WriteFile(fileName, []byte(testBody), 0666)
	}()

	f, err := OpenFileWithRetry(fileName, 50*time.Millisecond, 2*time.Second)
	if err != nil {
	}
	defer f.Close()

	body := make([]byte, 19)
	_, _ = f.Read(body)

	_ = os.Remove(fileName)

	if string(body) != testBody {
		t.Fail()
	}
}
