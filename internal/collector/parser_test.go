package collector

import "testing"

func TestGetIntStatus(t *testing.T) {
	acceptableStatuses := []string{"BAD", "GOOD"}

	for expected, status := range acceptableStatuses {
		result, ok := getIntStatus(status)
		if !ok || result != expected {
			t.Fail()
		}
	}

	result, ok := getIntStatus("NOTGOOD")
	if ok || result != -1 {
		t.Fail()
	}
}
