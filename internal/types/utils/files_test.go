package utils

import (
	"bytes"
	"io"
	"os"
	"path"
	"testing"
	"time"
)

func TestOpenFileWithRetry(t *testing.T) {
	t.Parallel()

	fileName := path.Join(t.TempDir(), t.Name())
	testBody := []byte(t.Name())

	go func(fileName string, testBody []byte) {
		time.Sleep(100 * time.Millisecond)

		_ = os.WriteFile(fileName, testBody, 0o600)
	}(fileName, testBody)

	f, err := OpenFileWithRetry(fileName)
	if err != nil {
		t.Fail()
	}

	defer f.Close()

	if body, _ := io.ReadAll(f); !bytes.Equal(body, testBody) {
		t.Fail()
	}
}
