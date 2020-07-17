package collector

import (
	"os"
	"testing"
)

func TestGetIntStatus(t *testing.T) {
	acceptableStatuses := []string{"BAD", "GOOD"}
	script := VRRPScript{}

	for expected, status := range acceptableStatuses {
		script.Status = status
		result, ok := script.getIntStatus()
		if !ok || result != expected {
			t.Fail()
		}
	}

	script.Status = "NOTGOOD"
	result, ok := script.getIntStatus()
	if ok || result != -1 {
		t.Fail()
	}
}

func TestGetIntState(t *testing.T) {
	acceptableStates := []string{"idle", "running", "requested termination", "forcing termination"}
	script := VRRPScript{}

	for expected, state := range acceptableStates {
		script.State = state
		result, ok := script.getIntState()
		if !ok || result != expected {
			t.Fail()
		}
	}

	script.State = "NOTGOOD"
	result, ok := script.getIntState()
	if ok || result != -1 {
		t.Fail()
	}
}

func TestGetStringState(t *testing.T) {
	acceptableStates := []string{"INIT", "BACKUP", "MASTER", "FAULT"}
	data := VRRPData{}

	for state, expected := range acceptableStates {
		data.State = state
		result, ok := data.getStringState()
		if !ok || result != expected {
			t.Fail()
		}
	}

	data.State = -1
	result, ok := data.getStringState()
	if ok || result != "" {
		t.Fail()
	}

	data.State = len(acceptableStates)
	result, ok = data.getStringState()
	if ok || result != "" {
		t.Fail()
	}
}

func TestVRRPDataStringToIntState(t *testing.T) {
	acceptableStates := []string{"INIT", "BACKUP", "MASTER", "FAULT"}

	for expected, state := range acceptableStates {
		result, ok := vrrpDataStringToIntState(state)
		if !ok || result != expected {
			t.Fail()
		}
	}

	result, ok := vrrpDataStringToIntState("NOGOOD")
	if ok || result != -1 {
		t.Fail()
	}
}

func TestParseVRRPData(t *testing.T) {
	f, err := os.Open("../../test_files/keepalived.data")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	defer f.Close()

	k := &KeepalivedCollector{}
	vrrpData, err := k.parseVRRPData(f)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if len(vrrpData) != 3 {
		t.Fail()
	}

	for _, data := range vrrpData {
		if data.IName == "VI_EXT_1" {
			if data.State != 2 {
				t.Fail()
			}
			if data.WantState != 2 {
				t.Fail()
			}
			if data.Intf != "ens192" {
				t.Fail()
			}
			if data.GArpDelay != 5 {
				t.Fail()
			}
			if data.VRID != 10 {
				t.Fail()
			}
			for _, ip := range data.VIPs {
				if ip != "192.168.2.1" {
					t.Fail()
				}
			}
		} else if data.IName == "VI_EXT_2" {
			if data.State != 1 {
				t.Fail()
			}
			if data.WantState != 1 {
				t.Fail()
			}
			if data.Intf != "ens192" {
				t.Fail()
			}
			if data.GArpDelay != 5 {
				t.Fail()
			}
			if data.VRID != 20 {
				t.Fail()
			}
			for _, ip := range data.VIPs {
				if ip != "192.168.2.2" {
					t.Fail()
				}
			}
		} else if data.IName == "VI_EXT_2" {
			if data.State != 1 {
				t.Fail()
			}
			if data.WantState != 1 {
				t.Fail()
			}
			if data.Intf != "ens192" {
				t.Fail()
			}
			if data.GArpDelay != 5 {
				t.Fail()
			}
			if data.VRID != 30 {
				t.Fail()
			}
			for _, ip := range data.VIPs {
				if ip != "192.168.2.3" {
					t.Fail()
				}
			}
		}
	}
}

func TestParseVRRPScript(t *testing.T) {
	f, err := os.Open("../../test_files/keepalived.data")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	defer f.Close()

	k := &KeepalivedCollector{}
	vrrpScripts := k.parseVRRPScript(f)

	if len(vrrpScripts) != 1 {
		t.Fail()
	}

	for _, script := range vrrpScripts {
		if script.Name != "check_script" {
			t.Fail()
		}
		if script.Status != "GOOD" {
			t.Fail()
		}
		if script.State != "idle" {
			t.Fail()
		}
	}
}
