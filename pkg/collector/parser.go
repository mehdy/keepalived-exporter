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

var VRRPScriptStatuses = []string{"BAD", "GOOD"}
var VRRPScriptStates = []string{"idle", "running", "requested termination", "forcing termination"}
var VRRPStates = []string{"INIT", "BACKUP", "MASTER", "FAULT"}

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
	if len(VRRPStates) <= state {
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

func (k *KeepalivedCollector) stats() (*KeepalivedStats, error) {
	stats := &KeepalivedStats{
		VRRPs:   make([]VRRP, 0),
		Scripts: make([]VRRPScript, 0),
	}

	if k.useJSON {
		err := k.signal(k.SIGJSON)
		if err != nil {
			logrus.Error("Failed to send JSON signal to keepalived: ", err)
			return nil, err
		}

		f, err := os.Open("/tmp/keepalived.json")
		if err != nil {
			logrus.Error("Failed to open /tmp/keepalived.json: ", err)
			return nil, err
		}
		defer f.Close()

		stats.VRRPs, err = k.parseJSON(f)
		if err != nil {
			logrus.Error("Failed to decode keepalived.json to VRRPStats array structure: ", err)
			return nil, err
		}
	} else {
		err := k.signal(k.SIGSTATS)
		if err != nil {
			logrus.Error("Failed to send STATS signal to keepalived: ", err)
			return nil, err
		}
		f, err := os.Open("/tmp/keepalived.stats")
		if err != nil {
			logrus.Error("Failed to open /tmp/keepalived.stats: ", err)
			return nil, err
		}
		vrrpStats, err := k.parseStats(f)
		if err != nil {
			return nil, err
		}
		f.Close()

		err = k.signal(k.SIGDATA)
		if err != nil {
			logrus.Error("Failed to send DATA signal to keepalived", " err: ", err)
			return nil, err
		}

		f, err = os.Open("/tmp/keepalived.data")
		if err != nil {
			logrus.Error("Failed to open /tmp/keepalived.data: ", err)
			return nil, err
		}
		defer f.Close()

		vrrpData, err := k.parseVRRPData(f)
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

		stats.Scripts = k.parseVRRPScript(f)
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
		} else if strings.HasPrefix(l, "   ") && strings.Contains(l, prop) && d.IName != "" {
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
				if err := d.parseVIPs(val, scanner); err != nil {
					return data, err
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

func (v *VRRPData) parseVIPs(VIPNums string, scanner *bufio.Scanner) error {
	vipNums, err := strconv.Atoi(VIPNums)
	if err != nil {
		logrus.Error("Failed to convert string to int in parseVIPS VIPNums: ", VIPNums, " err: ", err)
		return err
	}

	for i := 0; i < vipNums; i++ {
		l := scanner.Text()
		l = strings.TrimSpace(l)
		v.VIPs = append(v.VIPs, l)
	}

	return nil
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
				logrus.Error("Unknown metric value from keepalived.stats for key: ", key, " value: ", val, " err: ", err)
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
				logrus.Error("Unknown metric value from keepalived.stats for key: ", key, " value: ", val, " err: ", err)
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
