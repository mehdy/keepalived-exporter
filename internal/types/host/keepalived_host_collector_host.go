package host

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"

	"github.com/cafebazaar/keepalived-exporter/internal/collector"
	"github.com/cafebazaar/keepalived-exporter/internal/types/utils"
)

// KeepalivedHostCollectorHost implements Collector for when Keepalived and Keepalived Exporter are both on a same host
type KeepalivedHostCollectorHost struct {
	pidPath string
	version *version.Version
	useJSON bool

	SIGJSON  syscall.Signal
	SIGDATA  syscall.Signal
	SIGSTATS syscall.Signal
}

// NewKeepalivedHostCollectorHost is creating new instance of KeepalivedHostCollectorHost
func NewKeepalivedHostCollectorHost(useJSON bool, pidPath string) *KeepalivedHostCollectorHost {
	k := &KeepalivedHostCollectorHost{
		useJSON: useJSON,
		pidPath: pidPath,
	}

	var err error
	k.version, err = k.getKeepalivedVersion()
	if err != nil {
		logrus.WithError(err).Warn("Version detection failed. Assuming it's the latest one.")
	}

	k.initSignals()

	return k
}

func (k *KeepalivedHostCollectorHost) initSignals() {
	if k.useJSON {
		k.SIGJSON = k.sigNum("JSON")
	}
	k.SIGDATA = k.sigNum("DATA")
	k.SIGSTATS = k.sigNum("STATS")
}

// GetKeepalivedVersion returns Keepalived version
func (k *KeepalivedHostCollectorHost) getKeepalivedVersion() (*version.Version, error) {
	cmd := exec.Command("bash", "-c", "keepalived -v")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		logrus.WithFields(logrus.Fields{"stderr": stderr.String(), "stdout": stdout.String()}).WithError(err).Error("Error getting keepalived version")
		return nil, errors.New("Error getting keepalived version")
	}

	return utils.ParseVersion(stdout.String())
}

// Signal sends signal to Keepalived process
func (k *KeepalivedHostCollectorHost) signal(signal os.Signal) error {
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

// SigNum returns signal number for given signal name
func (k *KeepalivedHostCollectorHost) sigNum(sigString string) syscall.Signal {
	if !utils.HasSigNumSupport(k.version) {
		return utils.GetDefaultSignal(sigString)
	}

	sigNumCommand := "keepalived --signum=" + sigString
	cmd := exec.Command("bash", "-c", sigNumCommand)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "stderr": stderr.String()}).WithError(err).Fatal("Error getting signum")
	}

	signum, err := strconv.ParseInt(stdout.String(), 10, 32)
	if err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "signum": stdout.String()}).WithError(err).Fatal("Error parsing signum result")
	}

	return syscall.Signal(signum)
}

func (k *KeepalivedHostCollectorHost) JSONVrrps() ([]collector.VRRP, error) {
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

	return collector.ParseJSON(f)
}

func (k *KeepalivedHostCollectorHost) StatsVrrps() (map[string]*collector.VRRPStats, error) {
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

	return collector.ParseStats(f)
}

func (k *KeepalivedHostCollectorHost) DataVrrps() (map[string]*collector.VRRPData, error) {
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

	return collector.ParseVRRPData(f)
}

func (k *KeepalivedHostCollectorHost) ScriptVrrps() ([]collector.VRRPScript, error) {
	f, err := os.Open("/tmp/keepalived.data")
	if err != nil {
		logrus.WithError(err).Error("Failed to open /tmp/keepalived.data")
		return nil, err
	}
	defer f.Close()

	return collector.ParseVRRPScript(f), nil
}
