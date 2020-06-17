package collector

import "testing"

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
