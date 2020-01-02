package collector

import (
	"bufio"
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

func (v *VRRPScript) string2status(status string) (int, bool) {
	const (
		Bad = iota
		Good
	)

	switch status {
	case "BAD":
		return Bad, true
	case "GOOD":
		return Good, true
	}

	return -1, false
}

func (v *VRRPScript) string2state(state string) (int, bool) {
	const (
		Idle = iota
		Running
		RequestedTermination
		ForcingTermination
	)

	switch state {
	case "idle":
		return Idle, true
	case "running":
		return Running, true
	case "requested termination":
		return RequestedTermination, true
	case "forcing termination":
		return ForcingTermination, true
	}

	return -1, false
}

func (v *VRRPData) state2string(state int) (string, bool) {
	const (
		Init = iota
		Backup
		Master
		Fault
	)

	switch state {
	case Init:
		return "INIT", true
	case Backup:
		return "BACKUP", true
	case Master:
		return "MASTER", true
	case Fault:
		return "FAULT", true
	}

	return "", false
}

func (v *VRRPData) string2state(state string) (int, bool) {
	const (
		Init = iota
		Backup
		Master
		Fault
	)

	switch state {
	case "INIT":
		return Init, true
	case "BACKUP":
		return Backup, true
	case "MASTER":
		return Master, true
	case "FAULT":
		return Fault, true
	}

	return -1, false
}

func (k *KCollector) parseJSON() ([]Stats, error) {
	stats := make([]Stats, 0)

	f, err := os.Open("/tmp/keepalived.json")
	if err != nil {
		logrus.Error("Failed to open /tmp/keepalived.json", " err: ", err)
		return stats, err
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&stats)
	if err != nil {
		logrus.Error("Failed to decode keepalived.json to VRRPStats array structure", " err: ", err)
		return stats, err
	}

	return stats, nil
}

func (k *KCollector) parseVRRPData() ([]VRRPData, error) {
	data := make([]VRRPData, 0)

	f, err := os.Open("/tmp/keepalived.data")
	if err != nil {
		logrus.Error("Failed on opening /tmp/keepalived.data", "err: ", err)
		return data, err
	}
	defer f.Close()

	sep := "VRRP Instance"
	prop := "="

	d := VRRPData{}
	scanner := bufio.NewScanner(bufio.NewReader(f))

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

func (k *KCollector) parseVRRPScript() ([]VRRPScript, error) {
	scripts := make([]VRRPScript, 0)

	f, err := os.Open("/tmp/keepalived.data")
	if err != nil {
		logrus.Error("Failed on opening /tmp/keepalived.data", "err: ", err)
		return scripts, err
	}
	defer f.Close()

	sep := "VRRP Script"
	prop := "="

	script := VRRPScript{}
	scanner := bufio.NewScanner(bufio.NewReader(f))

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

	return scripts, nil
}

func (k *KCollector) parseStats() ([]VRRPStats, error) {
	stats := make([]VRRPStats, 0)

	f, err := os.Open("/tmp/keepalived.stats")
	if err != nil {
		logrus.Error("Failed to open /tmp/keepalived.stats", " err: ", err)
		return stats, err
	}
	defer f.Close()

	sep := "VRRP Instance"
	prop := ":"

	s := VRRPStats{}
	scanner := bufio.NewScanner(bufio.NewReader(f))

	section := ""
	instance := ""

	for scanner.Scan() {
		l := scanner.Text()
		if strings.HasPrefix(l, sep) && strings.Contains(l, prop) {
			if instance != "" {
				stats = append(stats, s)
				s = VRRPStats{}
				instance = ""
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
