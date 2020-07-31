package collector

import (
	"bytes"
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

func sigNum(sig string) int {
	sigNumCommand := "keepalived --signum=" + sig
	cmd := exec.Command("bash", "-c", sigNumCommand)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logrus.WithFields(logrus.Fields{"signal": sig, "stderr": stderr.String()}).WithError(err).Fatal("Error getting signum")
	}

	var signum int
	err = json.Unmarshal(stdout.Bytes(), &signum)
	if err != nil {
		logrus.WithFields(logrus.Fields{"signal": sig, "signum": stdout.String()}).WithError(err).Fatal("Error unmarshalling signum result")
	}

	return signum
}

func (k *KeepalivedCollector) signal(signal int) error {
	data, err := ioutil.ReadFile(k.pidPath)
	if err != nil {
		logrus.WithField("path", k.pidPath).WithError(err).Error("Can't find keepalived")
		return err
	}

	pid, err := strconv.Atoi(strings.TrimSuffix(string(data), "\n"))
	if err != nil {
		logrus.WithField("path", k.pidPath).WithError(err).Error("Unknown pid found for keepalived")
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		logrus.WithField("pid", pid).WithError(err).Error("Failed to find process")
		return err
	}

	err = proc.Signal(syscall.Signal(signal))
	if err != nil {
		logrus.WithField("pid", pid).WithError(err).Error("Failed to send signal")
		return err
	}

	// Wait 10ms for Keepalived to create its files
	time.Sleep(10 * time.Millisecond)
	return nil
}
