package types

import (
	"bytes"
	"os"
)

// KeepalivedCollector is an interface for implementing multiple collector modes
type KeepalivedCollector interface {
	SigNum(sig string) bytes.Buffer
	GetKeepalivedVersion() (*bytes.Buffer, error)
	Signal(signal os.Signal) error
}
