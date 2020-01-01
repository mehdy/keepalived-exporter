package collector

import (
	"encoding/json"
	"os"

	"github.com/sirupsen/logrus"
)

//States
const (
	Init = iota
	Backup
	Master
	Fault
)

var string2state = map[string]int{
	"INIT":   Init,
	"BACKUP": Backup,
	"MASTER": Master,
	"FAULT":  Fault,
}

var state2string = map[int]string{
	Init:   "INIT",
	Backup: "BACKUP",
	Master: "MASTER",
	Fault:  "FAULT",
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
		logrus.Error("Failed to decode keepalived.json to KStats array structure", " err: ", err)
		return stats, err
	}

	return stats, nil
}
