package host

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/mehdy/keepalived-exporter/internal/collector"
	"github.com/mehdy/keepalived-exporter/internal/types/utils"
	"github.com/sirupsen/logrus"
)

// KeepalivedHostCollectorHost implements Collector for when Keepalived and Keepalived Exporter are both on a same host.
type KeepalivedHostCollectorHost struct {
	pidPath string
	version *version.Version
	useJSON bool

	SIGJSON  syscall.Signal
	SIGDATA  syscall.Signal
	SIGSTATS syscall.Signal
}

// NewKeepalivedHostCollectorHost is creating new instance of KeepalivedHostCollectorHost.
func NewKeepalivedHostCollectorHost(useJSON bool, pidPath string) (*KeepalivedHostCollectorHost, error) {
	k := &KeepalivedHostCollectorHost{
		useJSON: useJSON,
		pidPath: pidPath,
	}

	var err error
	if k.version, err = k.getKeepalivedVersion(); err != nil {
		logrus.WithError(err).Warn("Version detection failed. Assuming it's the latest one.")
	}
	
	if useJSON {
		err = isEnableJSONSupported()
        	if err != nil {
			logrus.WithError(err).Warn("JSON support detection failed. Please check keepalivedJSON flag")
			return nil, err
        	}
	}
	
	k.initSignals()

	return k, nil
}

func (k *KeepalivedHostCollectorHost) Refresh() error {
	if k.useJSON {
		if err := k.signal(k.SIGJSON); err != nil {
			logrus.WithError(err).Error("Failed to send JSON signal to keepalived")

			return err
		}

		return nil
	}

	if err := k.signal(k.SIGSTATS); err != nil {
		logrus.WithError(err).Error("Failed to send STATS signal to keepalived")

		return err
	}

	if err := k.signal(k.SIGDATA); err != nil {
		logrus.WithError(err).Error("Failed to send DATA signal to keepalived")

		return err
	}

	return nil
}

func (k *KeepalivedHostCollectorHost) initSignals() {
	if k.useJSON {
		k.SIGJSON = k.sigNum("JSON")
	}

	k.SIGDATA = k.sigNum("DATA")
	k.SIGSTATS = k.sigNum("STATS")
}

// GetKeepalivedVersion returns Keepalived version.
func (k *KeepalivedHostCollectorHost) getKeepalivedVersion() (*version.Version, error) {
	var stdout, stderr bytes.Buffer

	cmd := exec.Command("bash", "-c", "keepalived -v")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logrus.WithFields(logrus.Fields{"stderr": stderr.String(), "stdout": stdout.String()}).
			WithError(err).
			Error("Error getting keepalived version")

		return nil, errors.New("error getting keepalived version")
	}

	return utils.ParseVersion(stderr.String())
}

func isEnableJSONSupported() error {
        cmd := exec.Command("keepalived", "--version")
        output, err := cmd.CombinedOutput()
        if err != nil {
                return fmt.Errorf("failed to execute keepalived --version: %v", err)
        }

        outputStr := string(output)

        if strings.Contains(outputStr, "--enable-json") {
                return nil
        }

        return fmt.Errorf("keepalived does not turn on the enable-json switch")
}

// Signal sends signal to Keepalived process.
func (k *KeepalivedHostCollectorHost) signal(signal os.Signal) error {
	data, err := os.ReadFile(k.pidPath)
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

	return nil
}

// SigNum returns signal number for given signal name.
func (k *KeepalivedHostCollectorHost) sigNum(sigString string) syscall.Signal {
	if !utils.HasSigNumSupport(k.version) {
		return utils.GetDefaultSignal(sigString)
	}

	var stdout, stderr bytes.Buffer

	sigNumCommand := "keepalived --signum=" + sigString
	cmd := exec.Command("bash", "-c", sigNumCommand)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "stderr": stderr.String()}).
			WithError(err).
			Fatal("Error getting signum")
	}

	return syscall.Signal(parseSigNum(stdout, sigString))
}

func (k *KeepalivedHostCollectorHost) JSONVrrps() ([]collector.VRRP, error) {
	const fileName = "/tmp/keepalived.json"

	f, err := os.Open(fileName)
	if err != nil {
		logrus.WithError(err).WithField("fileName", fileName).Error("failed to open JSON VRRP file")

		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logrus.WithError(err).Error("Failed to close file")
		}
	}()

	return collector.ParseJSON(f)
}

func (k *KeepalivedHostCollectorHost) StatsVrrps() (map[string]*collector.VRRPStats, error) {
	const fileName = "/tmp/keepalived.stats"

	f, err := os.Open(fileName)
	if err != nil {
		logrus.WithError(err).WithField("fileName", fileName).Error("failed to open Stats VRRP file")

		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logrus.WithError(err).Error("Failed to close file")
		}
	}()

	return collector.ParseStats(f)
}

func (k *KeepalivedHostCollectorHost) DataVrrps() (map[string]*collector.VRRPData, error) {
	const fileName = "/tmp/keepalived.data"

	f, err := os.Open(fileName)
	if err != nil {
		logrus.WithError(err).WithField("fileName", fileName).Error("failed to open Data VRRP file")

		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logrus.WithError(err).Error("Failed to close file")
		}
	}()

	return collector.ParseVRRPData(f)
}

func (k *KeepalivedHostCollectorHost) ScriptVrrps() ([]collector.VRRPScript, error) {
	const fileName = "/tmp/keepalived.data"

	f, err := os.Open(fileName)
	if err != nil {
		logrus.WithError(err).WithField("fileName", fileName).Error("failed to open Script VRRP file")

		return nil, err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logrus.WithError(err).Error("Failed to close file")
		}
	}()

	return collector.ParseVRRPScript(f), nil
}

// HasVRRPScriptStateSupport check if Keepalived version supports VRRP Script State in output.
func (k *KeepalivedHostCollectorHost) HasVRRPScriptStateSupport() bool {
	return utils.HasVRRPScriptStateSupport(k.version)
}
