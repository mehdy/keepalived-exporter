package collector

import (
	"reflect"
	"testing"
)

func TestSetState(t *testing.T) {
	data := VRRPData{}
	acceptableStates := []string{"INIT", "BACKUP", "MASTER", "FAULT"}

	for expected, state := range acceptableStates {
		err := data.setState(state)
		if err != nil || data.State != expected {
			t.Fail()
		}
	}

	err := data.setState("NOGOOD")
	if err == nil || data.State != -1 {
		t.Fail()
	}
}

func TestSetWantState(t *testing.T) {
	data := VRRPData{}
	acceptableStates := []string{"INIT", "BACKUP", "MASTER", "FAULT"}

	for expected, state := range acceptableStates {
		err := data.setWantState(state)
		if err != nil || data.WantState != expected {
			t.Fail()
		}
	}

	err := data.setWantState("NOGOOD")
	if err == nil || data.WantState != -1 {
		t.Fail()
	}
}

func TestSetGArpDelay(t *testing.T) {
	data := VRRPData{}

	delay := "1"
	expected := 1
	err := data.setGArpDelay(delay)
	if err != nil || data.GArpDelay != expected {
		t.Fail()
	}

	delay = "1.1"
	expected = 0
	err = data.setGArpDelay(delay)
	if err == nil || data.GArpDelay != expected {
		t.Fail()
	}

	delay = "NA"
	expected = 0
	err = data.setGArpDelay(delay)
	if err == nil || data.GArpDelay != expected {
		t.Fail()
	}
}

func TestSetVRID(t *testing.T) {
	data := VRRPData{}

	vrid := "10"
	expected := 10
	err := data.setVRID(vrid)
	if err != nil || data.VRID != expected {
		t.Fail()
	}

	vrid = "1.1"
	expected = 0
	err = data.setVRID(vrid)
	if err == nil || data.VRID != expected {
		t.Fail()
	}

	vrid = "NA"
	expected = 0
	err = data.setVRID(vrid)
	if err == nil || data.VRID != expected {
		t.Fail()
	}
}

func TestAddVIP(t *testing.T) {
	data := VRRPData{}

	vips := []string{"   1.1.1.1", "2.2.2.2", "3.3.3.3   "}
	expectedVIPs := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}

	for _, vip := range vips {
		data.addVIP(vip)
	}

	if !reflect.DeepEqual(expectedVIPs, data.VIPs) {
		t.Fail()
	}
}
