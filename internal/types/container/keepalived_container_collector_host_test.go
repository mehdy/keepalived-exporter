package container

import (
	"testing"

	"github.com/hashicorp/go-version"
)

func TestInitPaths(t *testing.T) {
	t.Parallel()

	k := KeepalivedContainerCollectorHost{}
	k.initPaths("/custom-tmp")

	if k.jsonPath != "/custom-tmp/keepalived.json" {
		t.Fail()
	}

	if k.statsPath != "/custom-tmp/keepalived.stats" {
		t.Fail()
	}

	if k.dataPath != "/custom-tmp/keepalived.data" {
		t.Fail()
	}
}

func TestHasVRRPScriptStateSupport(t *testing.T) {
	t.Parallel()

	notSupportingVersion := version.Must(version.NewVersion("1.3.5"))
	supportingVersion := version.Must(version.NewVersion("1.4.0"))

	c := KeepalivedContainerCollectorHost{
		version: notSupportingVersion,
	}
	if c.HasVRRPScriptStateSupport() {
		t.Fail()
	}

	c = KeepalivedContainerCollectorHost{
		version: supportingVersion,
	}
	if !c.HasVRRPScriptStateSupport() {
		t.Fail()
	}

	c = KeepalivedContainerCollectorHost{}
	if !c.HasVRRPScriptStateSupport() {
		t.Fail()
	}
}
