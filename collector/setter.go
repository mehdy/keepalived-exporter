package collector

import (
	"errors"
	"strconv"

	"github.com/sirupsen/logrus"
)

func (v *VRRPData) setState(state string) error {
	ok := true
	if v.State, ok = v.string2state(state); !ok {
		logrus.Error("Unknown state found: ", state, " iname: ", v.IName)
		return errors.New("Unknown state found: " + state + " iname: " + v.IName)
	}

	return nil
}

func (v *VRRPData) setWantState(wantState string) error {
	ok := true
	if v.WantState, ok = v.string2state(wantState); !ok {
		logrus.Error("Unknown wantstate found: ", wantState)
		return errors.New("Unknown wantstate found: " + wantState)
	}

	return nil
}

func (v *VRRPData) setGArpDelay(delay string) error {
	var err error
	if v.GArpDelay, err = strconv.Atoi(delay); err != nil {
		logrus.Error("Failed to parse GArpDelay to int delay: ", delay, " err: ", err)
		return err
	}

	return nil
}

func (v *VRRPData) setVRID(vrid string) error {
	var err error
	if v.VRID, err = strconv.Atoi(vrid); err != nil {
		logrus.Error("Failed to parse vrid to int vrid: ", vrid, " err: ", err)
		return err
	}

	return nil
}
