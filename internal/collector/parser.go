package collector

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	// VRRPScriptStatuses contains VRRP Script statuses
	VRRPScriptStatuses = []string{"BAD", "GOOD"}
	// VRRPScriptStates contains VRRP Script states
	VRRPScriptStates = []string{"idle", "running", "requested termination", "forcing termination"}
	// VRRPStates contains VRRP states
	VRRPStates = []string{"INIT", "BACKUP", "MASTER", "FAULT"}
)

func (VRRPScript) getIntStatus(status string) (int, bool) {
	for i, s := range VRRPScriptStatuses {
		if s == status {
			return i, true
		}
	}
	return -1, false
}

func (VRRPScript) getIntState(state string) (int, bool) {
	for i, s := range VRRPScriptStates {
		if s == state {
			return i, true
		}
	}
	return -1, false
}

func (VRRPData) getStringState(state int) (string, bool) {
	if state < len(VRRPStates) && state >= 0 {
		return VRRPStates[state], true
	}
	return "", false
}

func (VRRPData) getIntState(state string) (int, bool) {
	for i, s := range VRRPStates {
		if s == state {
			return i, true
		}
	}
	return -1, false
}

func (k *KeepalivedCollector) jsonVrrps() ([]VRRP, error) {
	err := k.signal(k.SIGJSON)
	if err != nil {
		logrus.WithError(err).Error("Failed to send JSON signal to keepalived")
		return nil, err
	}

	f, err := os.Open("/tmp/keepalived.json")
	if err != nil {
		logrus.WithError(err).Error("Failed to open /tmp/keepalived.json")
		return nil, err
	}
	defer f.Close()

	VRRPs, err := k.parseJSON(f)
	if err != nil {
		logrus.WithError(err).Error("Failed to decode keepalived.json to VRRPStats array structure")
		return nil, err
	}

	return VRRPs, nil
}

func (k *KeepalivedCollector) statsVrrps() ([]VRRPStats, error) {
	err := k.signal(k.SIGSTATS)
	if err != nil {
		logrus.WithError(err).Error("Failed to send STATS signal to keepalived")
		return nil, err
	}

	f, err := os.Open("/tmp/keepalived.stats")
	if err != nil {
		logrus.WithError(err).Error("Failed to open /tmp/keepalived.stats")
		return nil, err
	}
	defer f.Close()

	vrrpStats, err := k.parseStats(f)
	if err != nil {
		return nil, err
	}

	return vrrpStats, nil
}

func (k *KeepalivedCollector) dataVrrps() ([]VRRPData, error) {
	err := k.signal(k.SIGDATA)
	if err != nil {
		logrus.WithError(err).Error("Failed to send DATA signal to keepalived")
		return nil, err
	}

	f, err := os.Open("/tmp/keepalived.data")
	if err != nil {
		logrus.WithError(err).Error("Failed to open /tmp/keepalived.data")
		return nil, err
	}
	defer f.Close()

	vrrpData, err := k.parseVRRPData(f)
	if err != nil {
		return nil, err
	}

	return vrrpData, nil
}

func (k *KeepalivedCollector) scriptVrrps() ([]VRRPScript, error) {
	f, err := os.Open("/tmp/keepalived.data")
	if err != nil {
		logrus.WithError(err).Error("Failed to open /tmp/keepalived.data")
		return nil, err
	}
	defer f.Close()

	return k.parseVRRPScript(f), nil
}

func (k *KeepalivedCollector) stats() (*KeepalivedStats, error) {
	stats := &KeepalivedStats{
		VRRPs:   make([]VRRP, 0),
		Scripts: make([]VRRPScript, 0),
	}
	var err error

	if k.useJSON {
		stats.VRRPs, err = k.jsonVrrps()
		if err != nil {
			return nil, err
		}
	} else {
		vrrpStats, err := k.statsVrrps()
		if err != nil {
			return nil, err
		}

		vrrpData, err := k.dataVrrps()
		if err != nil {
			return nil, err
		}

		if len(vrrpData) != len(vrrpStats) {
			logrus.Error("keepalived.data and keepalived.stats datas are not synced")
			return nil, errors.New("keepalived.data and keepalived.stats datas are not synced")
		}

		for i := 0; i < len(vrrpData); i++ {
			s := VRRP{
				Data:  vrrpData[i],
				Stats: vrrpStats[i],
			}
			stats.VRRPs = append(stats.VRRPs, s)
		}

		stats.Scripts, err = k.scriptVrrps()
		if err != nil {
			return nil, err
		}
	}

	return stats, nil
}

func (k *KeepalivedCollector) parseJSON(i io.Reader) ([]VRRP, error) {
	stats := make([]VRRP, 0)

	err := json.NewDecoder(i).Decode(&stats)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func (k *KeepalivedCollector) parseVRRPData(i io.Reader) ([]VRRPData, error) {
	data := make([]VRRPData, 0)

	sep := "VRRP Instance"
	prop := "="

	d := VRRPData{}
	scanner := bufio.NewScanner(bufio.NewReader(i))

	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, " "+sep) && strings.Contains(l, prop) {
			if d.IName != "" {
				data = append(data, d)
				d = VRRPData{}
			}

			s := strings.Split(strings.TrimSpace(l), prop)
			d.IName = strings.TrimSpace(s[1])
		} else if strings.HasPrefix(l, "   ") && d.IName != "" {
			if !strings.Contains(l, prop) {
				continue
			}
			s := strings.Split(strings.TrimSpace(l), prop)
			key := strings.TrimSpace(s[0])
			val := strings.TrimSpace(s[1])
			switch key {
			case "State":
				if err := d.setState(val); err != nil {
					return data, err
				}
			case "Wantstate":
				if err := d.setWantState(val); err != nil {
					return data, err
				}
			case "Interface":
				d.Intf = val
			case "Gratuitous ARP delay":
				if err := d.setGArpDelay(val); err != nil {
					return data, err
				}
			case "Virtual Router ID":
				if err := d.setVRID(val); err != nil {
					return data, err
				}
			case "Virtual IP":
				vipNums, err := strconv.Atoi(val)
				if err != nil {
					logrus.WithField("VIPNums", val).WithError(err).Error("Failed to convert string to int in parseVIPS")
					return data, err
				}
				for i := 0; i < vipNums; i++ {
					if scanner.Scan() {
						vip := scanner.Text()
						d.setVIP(vip)
					} else {
						return data, scanner.Err()
					}
				}
			}
		} else {
			if d.IName != "" {
				data = append(data, d)
				d = VRRPData{}
			}
		}
	}

	if d.IName != "" {
		data = append(data, d)
		d = VRRPData{}
	}

	return data, nil
}

func (k *KeepalivedCollector) parseVRRPScript(i io.Reader) []VRRPScript {
	scripts := make([]VRRPScript, 0)

	sep := "VRRP Script"
	prop := "="

	script := VRRPScript{}
	scanner := bufio.NewScanner(bufio.NewReader(i))

	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, " "+sep) && strings.Contains(l, prop) {
			if script.Name != "" {
				scripts = append(scripts, script)
				script = VRRPScript{}
			}

			s := strings.Split(strings.TrimSpace(l), prop)
			script.Name = strings.TrimSpace(s[1])
		} else if strings.HasPrefix(l, "   ") && strings.Contains(l, prop) && script.Name != "" {
			s := strings.Split(strings.TrimSpace(l), prop)
			key := strings.TrimSpace(s[0])
			val := strings.TrimSpace(s[1])
			switch key {
			case "Status":
				script.Status = val
			case "State":
				script.State = val
			}
		} else if !strings.HasPrefix(l, "    ") {
			if script.Name != "" {
				scripts = append(scripts, script)
				script = VRRPScript{}
			}
		}
	}

	if script.Name != "" {
		scripts = append(scripts, script)
	}

	return scripts
}

func (k *KeepalivedCollector) parseStats(i io.Reader) ([]VRRPStats, error) {
	stats := make([]VRRPStats, 0)

	sep := "VRRP Instance"
	prop := ":"

	s := VRRPStats{}
	scanner := bufio.NewScanner(bufio.NewReader(i))

	var instance, section string

	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, sep) && strings.Contains(l, prop) {
			if instance != "" {
				stats = append(stats, s)
				s = VRRPStats{}
			}

			sp := strings.Split(strings.TrimSpace(l), prop)
			instance = strings.TrimSpace(sp[1])
		} else if strings.HasPrefix(l, "  ") && strings.HasSuffix(l, prop) {
			sp := strings.Split(strings.TrimSpace(l), prop)
			section = strings.TrimSpace(sp[0])
		} else if strings.HasPrefix(l, "    ") && section != "" {
			sp := strings.Split(strings.TrimSpace(l), prop)
			key := strings.TrimSpace(sp[0])
			val := strings.TrimSpace(sp[1])

			value, err := strconv.Atoi(val)
			if err != nil {
				logrus.WithFields(logrus.Fields{"key": key, "val": val}).WithError(err).Error("Unknown metric value from keepalived.stats")
				return stats, err
			}

			switch section {
			case "Advertisements":
				switch key {
				case "Received":
					s.AdvertRcvd = value
				case "Sent":
					s.AdvertSent = value
				}
			case "Packet Errors":
				switch key {
				case "Length":
					s.PacketLenErr = value
				case "TTL":
					s.IPTTLErr = value
				case "Invalid Type":
					s.InvalidTypeRcvd = value
				case "Advertisement Interval":
					s.AdvertIntervalErr = value
				case "Address List":
					s.AddrListErr = value
				}
			case "Authentication Errors":
				switch key {
				case "Invalid Type":
					s.InvalidAuthType = value
				case "Type Mismatch":
					s.AuthTypeMismatch = value
				case "Failure":
					s.AuthFailure = value
				}
			case "Priority Zero":
				switch key {
				case "Received":
					s.PRIZeroRcvd = value
				case "Sent":
					s.PRIZeroSent = value
				}
			}
		} else if strings.HasPrefix(l, "  ") && !strings.HasSuffix(l, prop) && !strings.HasPrefix(l, "    ") {
			sp := strings.Split(strings.TrimSpace(l), prop)
			key := strings.TrimSpace(sp[0])
			val := strings.TrimSpace(sp[1])
			section = ""

			value, err := strconv.Atoi(val)
			if err != nil {
				logrus.WithFields(logrus.Fields{"key": key, "val": val}).WithError(err).Error("Unknown metric value from keepalived.stats")
				return stats, err
			}

			switch key {
			case "Became master":
				s.BecomeMaster = value
			case "Released master":
				s.ReleaseMaster = value
			}
		}
	}

	if instance != "" {
		stats = append(stats, s)
	}

	return stats, nil
}
