package host

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// SigNum returns signal number for given signal name
func (khch *KeepalivedHostCollectorHost) SigNum(sig string) bytes.Buffer {
	sigNumCommand := "keepalived --signum=" + sig
	cmd := exec.Command("bash", "-c", sigNumCommand)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logrus.WithFields(logrus.Fields{"signal": sig, "stderr": stderr.String()}).WithError(err).Fatal("Error getting signum")
	}

	return stdout
}

// Signal sends signal to Keepalived process
func (khch *KeepalivedHostCollectorHost) Signal(signal os.Signal) error {
	data, err := ioutil.ReadFile(khch.pidPath)
	if err != nil {
		logrus.WithField("path", khch.pidPath).WithError(err).Error("Can't find keepalived")
		return err
	}

	pid, err := strconv.Atoi(strings.TrimSuffix(string(data), "\n"))
	if err != nil {
		logrus.WithField("path", khch.pidPath).WithError(err).Error("Unknown pid found for keepalived")
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		logrus.WithField("pid", pid).WithError(err).Error("Failed to find process")
		return err
	}

	err = proc.Signal(signal)
	if err != nil {
		logrus.WithField("pid", pid).WithError(err).Error("Failed to send signal")
		return err
	}

	// Wait 10ms for Keepalived to create its files
	time.Sleep(10 * time.Millisecond)
	return nil
}
