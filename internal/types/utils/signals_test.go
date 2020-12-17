package utils

import (
	"syscall"
	"testing"

	"github.com/hashicorp/go-version"
)

func TestHasSigNumSupport(t *testing.T) {
	notSupportingVersion := version.Must(version.NewVersion("1.3.5"))
	if HasSigNumSupport(notSupportingVersion) {
		t.Fail()
	}

	supportingVersion := version.Must(version.NewVersion("1.3.8"))
	if !HasSigNumSupport(supportingVersion) {
		t.Fail()
	}

	if !HasSigNumSupport(nil) {
		t.Fail()
	}
}

func TestGetDefaultSignal(t *testing.T) {
	dataSignal := syscall.SIGUSR1
	if GetDefaultSignal("DATA") != dataSignal {
		t.Fail()
	}

	statsSignal := syscall.SIGUSR2
	if GetDefaultSignal("STATS") != statsSignal {
		t.Fail()
	}
}
