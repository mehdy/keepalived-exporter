package collector

import (
	"os"
	"reflect"
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

func TestParseStats(t *testing.T) {
	f, err := os.Open("../../test_files/keepalived.stats")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	defer f.Close()

	k := &KeepalivedCollector{}
	stats, err := k.parseStats(f)
	if err != nil {
		t.Log(err)
		t.Fail()
	}

	if len(stats) != 3 {
		t.Fail()
	}

	// check for VI_EXT_1
	viExt1 := VRRPStats{
		AdvertRcvd:        11,
		AdvertSent:        12,
		BecomeMaster:      2,
		ReleaseMaster:     1,
		PacketLenErr:      1,
		IPTTLErr:          1,
		InvalidTypeRcvd:   1,
		AdvertIntervalErr: 1,
		AddrListErr:       1,
		InvalidAuthType:   2,
		AuthTypeMismatch:  2,
		AuthFailure:       2,
		PRIZeroRcvd:       1,
		PRIZeroSent:       1,
	}
	if !reflect.DeepEqual(viExt1, stats[0]) {
		t.Fail()
	}

	// check for VI_EXT_2
	viExt2 := VRRPStats{
		AdvertRcvd:        10,
		AdvertSent:        158,
		BecomeMaster:      2,
		ReleaseMaster:     2,
		PacketLenErr:      10,
		IPTTLErr:          10,
		InvalidTypeRcvd:   10,
		AdvertIntervalErr: 10,
		AddrListErr:       10,
		InvalidAuthType:   20,
		AuthTypeMismatch:  20,
		AuthFailure:       20,
		PRIZeroRcvd:       12,
		PRIZeroSent:       12,
	}
	if !reflect.DeepEqual(viExt2, stats[1]) {
		t.Fail()
	}

	// check for VI_EXT_3
	viExt3 := VRRPStats{
		AdvertRcvd:        23,
		AdvertSent:        172,
		BecomeMaster:      4,
		ReleaseMaster:     4,
		PacketLenErr:      30,
		IPTTLErr:          30,
		InvalidTypeRcvd:   30,
		AdvertIntervalErr: 30,
		AddrListErr:       30,
		InvalidAuthType:   10,
		AuthTypeMismatch:  10,
		AuthFailure:       2,
		PRIZeroRcvd:       1,
		PRIZeroSent:       2,
	}
	if !reflect.DeepEqual(viExt3, stats[2]) {
		t.Fail()
	}
}
