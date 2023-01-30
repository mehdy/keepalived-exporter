package collector

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func (v *VRRPData) setState(state string) error {
	var ok bool
	if v.State, ok = vrrpDataStringToIntState(state); !ok {
		logrus.WithFields(logrus.Fields{"state": state, "iname": v.IName}).Error("Unknown state found")

		return fmt.Errorf("unknown state found: %s, iname: %s", state, v.IName)
	}

	return nil
}

func (v *VRRPData) setWantState(wantState string) error {
	var ok bool
	if v.WantState, ok = vrrpDataStringToIntState(wantState); !ok {
		logrus.WithField("wantstate", wantState).Error("Unknown wantstate found")

		return fmt.Errorf("unknown wantstate found: %s", wantState)
	}

	return nil
}

func (v *VRRPData) setGArpDelay(delay string) error {
	var err error
	if v.GArpDelay, err = strconv.Atoi(delay); err != nil {
		logrus.WithField("delay", delay).WithError(err).Error("Failed to parse GArpDelay to int delay")

		return err
	}

	return nil
}

func (v *VRRPData) setVRID(vrid string) error {
	var err error
	if v.VRID, err = strconv.Atoi(vrid); err != nil {
		logrus.WithField("vrid", vrid).WithError(err).Error("Failed to parse vrid to int")

		return err
	}

	return nil
}

func (v *VRRPData) addVIP(vip string) {
	vip = strings.TrimSpace(vip)
	v.VIPs = append(v.VIPs, vip)
}
