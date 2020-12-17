package host

import (
	"testing"

	"github.com/hashicorp/go-version"
)

func TestHasVRRPScriptStateSupport(t *testing.T) {
	notSupportingVersion := version.Must(version.NewVersion("1.3.5"))
	c := KeepalivedHostCollectorHost{
		version: notSupportingVersion,
	}
	if c.HasVRRPScriptStateSupport() {
		t.Fail()
	}

	supportingVersion := version.Must(version.NewVersion("1.4.0"))
	c = KeepalivedHostCollectorHost{
		version: supportingVersion,
	}
	if !c.HasVRRPScriptStateSupport() {
		t.Fail()
	}

	c = KeepalivedHostCollectorHost{}
	if !c.HasVRRPScriptStateSupport() {
		t.Fail()
	}
}
