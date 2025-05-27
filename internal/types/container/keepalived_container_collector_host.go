package container

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/hashicorp/go-version"
	"github.com/mehdy/keepalived-exporter/internal/collector"
	"github.com/mehdy/keepalived-exporter/internal/types/utils"
	"github.com/sirupsen/logrus"
)

// KeepalivedContainerCollectorHost implements Collector for when Keepalived is on container and Keepalived Exporter is on a host.
type KeepalivedContainerCollectorHost struct {
	version       *version.Version
	useJSON       bool
	containerName string
	dataPath      string
	jsonPath      string
	statsPath     string
	dockerCli     *client.Client
	pidPath       string

	SIGJSON  syscall.Signal
	SIGDATA  syscall.Signal
	SIGSTATS syscall.Signal
}

// NewKeepalivedContainerCollectorHost is creating new instance of KeepalivedContainerCollectorHost.
func NewKeepalivedContainerCollectorHost(
	useJSON bool,
	containerName, containerTmpDir, pidPath string,
) *KeepalivedContainerCollectorHost {
	k := &KeepalivedContainerCollectorHost{
		useJSON:       useJSON,
		containerName: containerName,
		pidPath:       pidPath,
	}

	var err error

	k.dockerCli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		logrus.WithError(err).Fatal("Error creating docker env client")
	}

	k.version, err = k.getKeepalivedVersion()
	if err != nil {
		logrus.WithError(err).Warn("Version detection failed. Assuming it's the latest one.")
	}

	k.initSignals()

	k.initPaths(containerTmpDir)

	return k
}

func (k *KeepalivedContainerCollectorHost) Refresh() error {
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

func (k *KeepalivedContainerCollectorHost) initPaths(containerTmpDir string) {
	k.jsonPath = filepath.Join(containerTmpDir, "keepalived.json")
	k.statsPath = filepath.Join(containerTmpDir, "keepalived.stats")
	k.dataPath = filepath.Join(containerTmpDir, "keepalived.data")
}

// GetKeepalivedVersion returns Keepalived version.
func (k *KeepalivedContainerCollectorHost) getKeepalivedVersion() (*version.Version, error) {
	getVersionCmd := []string{"keepalived", "-v"}

	stdout, err := k.dockerExecCmd(getVersionCmd)
	if err != nil {
		return nil, err
	}

	return utils.ParseVersion(stdout.String())
}

func (k *KeepalivedContainerCollectorHost) initSignals() {
	if k.useJSON {
		k.SIGJSON = k.sigNum("JSON")
	}

	k.SIGDATA = k.sigNum("DATA")
	k.SIGSTATS = k.sigNum("STATS")
}

// SigNum returns signal number for given signal name.
func (k *KeepalivedContainerCollectorHost) sigNum(sigString string) syscall.Signal {
	if !utils.HasSigNumSupport(k.version) {
		return utils.GetDefaultSignal(sigString)
	}

	sigNumCommand := []string{"keepalived", "--signum", sigString}

	stdout, err := k.dockerExecCmd(sigNumCommand)
	if err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "container": k.containerName}).
			WithError(err).
			Fatal("Error getting signum")
	}

	reg := regexp.MustCompile("[^0-9]+")
	strSigNum := reg.ReplaceAllString(stdout.String(), "")

	signum, err := strconv.ParseInt(strSigNum, 10, 32)
	if err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "signum": stdout.String()}).
			WithError(err).
			Fatal("Error parsing signum result")
	}

	return syscall.Signal(signum)
}

func (k *KeepalivedContainerCollectorHost) dockerExecSignal(signal syscall.Signal) error {
	pidData, err := os.ReadFile(k.pidPath)
	if err != nil {
		logrus.WithField("path", k.pidPath).WithError(err).Error("Failed to read keepalived pid file")

		return err
	}

	pid := strings.TrimSpace(string(pidData))
	cmd := strslice.StrSlice{"kill", "-" + strconv.Itoa(int(signal)), pid}

	_, err = k.dockerExecCmd(cmd)

	return err
}

func (k *KeepalivedContainerCollectorHost) dockerSignal(signal syscall.Signal) error {
	err := k.dockerCli.ContainerKill(context.Background(), k.containerName, strconv.Itoa(int(signal)))
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"container": k.containerName,
			"signal":    int(signal),
		}).Error("Failed to send signal")

		return err
	}

	return nil
}

// Signal sends signal to Keepalived process.
func (k *KeepalivedContainerCollectorHost) signal(signal syscall.Signal) error {
	if k.pidPath != "" {
		return k.dockerExecSignal(signal)
	}

	return k.dockerSignal(signal)
}

// JSONVrrps send SIGJSON and parse the data to the list of collector.VRRP struct.
func (k *KeepalivedContainerCollectorHost) JSONVrrps() ([]collector.VRRP, error) {
	f, err := os.Open(k.jsonPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.jsonPath).Error("Failed to open keepalived.json")

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

// StatsVrrps send SIGSTATS and parse the stats.
func (k *KeepalivedContainerCollectorHost) StatsVrrps() (map[string]*collector.VRRPStats, error) {
	f, err := os.Open(k.statsPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.statsPath).Error("Failed to open keepalived.stats")

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

// DataVrrps send SIGDATA ans parse the data.
func (k *KeepalivedContainerCollectorHost) DataVrrps() (map[string]*collector.VRRPData, error) {
	f, err := os.Open(k.dataPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.dataPath).Error("Failed to open keepalived.data")

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

// ScriptVrrps parse the script data from keepalived.data.
func (k *KeepalivedContainerCollectorHost) ScriptVrrps() ([]collector.VRRPScript, error) {
	f, err := os.Open(k.dataPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.dataPath).Error("Failed to open keepalived.data")

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
func (k *KeepalivedContainerCollectorHost) HasVRRPScriptStateSupport() bool {
	return utils.HasVRRPScriptStateSupport(k.version)
}

func (k *KeepalivedContainerCollectorHost) HasJSONSignalSupport() (bool, error) {
	// exec command to check if SIGJSON is supported
	cmd := strslice.StrSlice{"keepalived", "--version"}
	output, err := k.dockerExecCmd(cmd)
	if err != nil {
		return false, err
	}

	if strings.Contains(output.String(), "--enable-json") {
		return true, nil
	}

	logrus.Error("Keepalived does not support JSON signal. Please check if it was compiled with --enable-json option")

	return false, nil
}
