package collector

import "testing"

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
