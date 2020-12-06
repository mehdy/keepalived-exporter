package container

import "testing"

func TestInitPaths(t *testing.T) {
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
