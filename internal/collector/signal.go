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

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

func isSigNumSupport() bool {
	v := keepalivedVersion()
	sigNumSupportedVersion, err := version.NewVersion("1.3.8")
	if err != nil {
		logrus.WithError(err).Fatal("Unexcpected error")
	}

	return v.GreaterThanOrEqual(sigNumSupportedVersion)
}

func sigNum(sig string) os.Signal {
	if !isSigNumSupport() {
		switch sig {
		case "DATA":
			return syscall.SIGUSR1
		case "STATS":
			return syscall.SIGUSR2
		default:
			logrus.WithField("signal", sig).Fatal("Unsupported signal for your keepalived")
		}
	}

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

	return syscall.Signal(signum)
}

func (k *KeepalivedCollector) signal(signal os.Signal) error {
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

	err = proc.Signal(signal)
	if err != nil {
		logrus.WithField("pid", pid).WithError(err).Error("Failed to send signal")
		return err
	}

	// Wait 10ms for Keepalived to create its files
	time.Sleep(10 * time.Millisecond)
	return nil
}
