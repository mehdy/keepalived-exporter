package collector

import (
	"errors"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func (v *VRRPData) setState(state string) error {
	var ok bool
	if v.State, ok = v.getIntState(state); !ok {
		logrus.WithFields(logrus.Fields{"state": state, "iname": v.IName}).Error("Unknown state found")
		return errors.New("Unknown state found: " + state + " iname: " + v.IName)
	}

	return nil
}

func (v *VRRPData) setWantState(wantState string) error {
	var ok bool
	if v.WantState, ok = v.getIntState(wantState); !ok {
		logrus.WithField("wantstate", wantState).Error("Unknown wantstate found")
		return errors.New("Unknown wantstate found: " + wantState)
	}

	return nil
}

func (v *VRRPData) setGArpDelay(delay string) error {
	var err error
	if v.GArpDelay, err = strconv.Atoi(delay); err != nil {
		logrus.WithField("delay", delay).Error("Failed to parse GArpDelay to int delay: ", err)
		return err
	}

	return nil
}

func (v *VRRPData) setVRID(vrid string) error {
	var err error
	if v.VRID, err = strconv.Atoi(vrid); err != nil {
		logrus.WithField("vrid", vrid).Error("Failed to parse vrid to int: ", err)
		return err
	}

	return nil
}

func (v *VRRPData) setVIP(vip string) {
	vip = strings.TrimSpace(vip)
	v.VIPs = append(v.VIPs, vip)
}
