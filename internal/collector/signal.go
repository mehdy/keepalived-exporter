package collector

import (
	"context"
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
	out, err := exec.Command("bash", "-c", sigNumCommand).Output()
	if err != nil {
		logrus.WithField("signal", sig).Fatal("Error getting signum: ", err)
	}

	var signum int
	err = json.Unmarshal(out, &signum)
	if err != nil {
		logrus.WithField("signal", sig).Fatal("Error unmarshalling signum result: ", err)
	}

	return signum
}

func (k *KeepalivedCollector) signal(signal int) error {
	data, err := ioutil.ReadFile(k.pidPath)
	if err != nil {
		logrus.WithField("path", k.pidPath).Error("Can't find keepalived: ", err)
		return err
	}

	pid, err := strconv.Atoi(strings.TrimSuffix(string(data), "\n"))
	if err != nil {
		logrus.WithField("path", k.pidPath).Error("Unknown pid found for keepalived: ", err)
		return err
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		logrus.WithField("pid", pid).Error("Failed to find process: ", err)
		return err
	}

	err = proc.Signal(syscall.Signal(signal))
	if err != nil {
		logrus.WithField("pid", pid).Error("Failed to send signal: ", err)
		return err
	}

	// Wait 10ms for Keepalived to create its files
	time.Sleep(10 * time.Millisecond)
	return nil
}

func (k *KeepalivedCollector) signalHandler(signum int) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	finishedSignal := make(chan bool)
	defer cancel()

	if !k.runningSignal {
		k.runningSignal = true
		go func() {
			err := k.signal(signum)
			if err != nil {
				logrus.WithField("SigNum", signum).Error("Failed to send signal to keepalived: ", err)
			}
			k.runningSignal = false
			finishedSignal <- true
		}()
	}

	select {
	case <-ctx.Done():
		logrus.WithField("SigNum", signum).Warn("Failed get response from signal less than 1 seconds")
		if signum == k.SIGSTATS {
			k.failedStatsSignal = true
		}
	case <-finishedSignal:
		logrus.WithField("SigNum", signum).Info("Signal process finished successfully")
		if signum == k.SIGSTATS {
			k.failedStatsSignal = false
		}
	}
}
