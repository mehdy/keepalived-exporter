package utils

import (
	"testing"

	"github.com/hashicorp/go-version"
)

func TestHasVRRPScriptStateSupport(t *testing.T) {
	t.Parallel()

	notSupportingVersion := version.Must(version.NewVersion("1.3.5"))
	if HasSigNumSupport(notSupportingVersion) {
		t.Fail()
	}

	supportingVersion := version.Must(version.NewVersion("1.4.0"))
	if !HasSigNumSupport(supportingVersion) {
		t.Fail()
	}

	if !HasSigNumSupport(nil) {
		t.Fail()
	}
}
