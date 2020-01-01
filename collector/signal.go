package collector

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

//Signals
var (
	SIGDATA  = sigNum("DATA")
	SIGSTATS = sigNum("STATS")
	SIGJSON  = sigNum("JSON")
)

func sigNum(sig string) int {
	out, err := exec.Command("bash", "-c", "keepalived --signum="+sig).Output()
	if err != nil {
		logrus.Fatal("Error getting signum for signal: ", sig, " err: ", err)
	}

	var signum int
	err = json.Unmarshal(out, &signum)
	if err != nil {
		logrus.Fatal("Error unmarshalling signum result for signal:", sig, " err: ", err)
	}

	return signum
}

func (k *KCollector) signal(sig syscall.Signal) error {
	data, err := ioutil.ReadFile("/var/run/keepalived.pid")
	if err != nil {
		logrus.Error("Can't find keepalived pid from /var/run/keepalived.pid", " err: ", err)
		return err
	}

	pid, err := strconv.Atoi(strings.TrimSuffix(string(data), "\n"))
	if err != nil {
		logrus.Error("Unknown pid found for keepalived from /var/run/keepalived.pid", " data: ", data, " err: ", err)
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		logrus.Error("Failed to find process with pid:", pid, " err: ", err)
		return err
	}

	err = proc.Signal(sig)
	if err != nil {
		logrus.Error("Failed to send signal to pid:", pid, " err: ", err)
		return err
	}

	time.Sleep(10 * time.Millisecond)
	return nil
}
