package container

import (
	"bytes"
	"context"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/cafebazaar/keepalived-exporter/internal/collector"
	"github.com/cafebazaar/keepalived-exporter/internal/types/utils"
	"github.com/docker/docker/client"
	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

// KeepalivedContainerCollectorHost implements Collector for when Keepalived is on container and Keepalived Exporter is on a host
type KeepalivedContainerCollectorHost struct {
	version       *version.Version
	useJSON       bool
	containerName string
	endpoint      *url.URL
	dataPath      string
	jsonPath      string
	statsPath     string
	dockerCli     *client.Client

	SIGJSON  syscall.Signal
	SIGDATA  syscall.Signal
	SIGSTATS syscall.Signal
}

// NewKeepalivedContainerCollectorHost is creating new instance of KeepalivedContainerCollectorHost
func NewKeepalivedContainerCollectorHost(useJSON bool, containerName, containerTmpDir, endpoint string) *KeepalivedContainerCollectorHost {
	k := &KeepalivedContainerCollectorHost{
		useJSON:       useJSON,
		containerName: containerName,
	}

	if endpoint != "" && containerName != "" {
		logrus.WithFields(logrus.Fields{"endpoint": endpoint, "containerName": containerName}).Fatal("Both container-name and endpoint can't be set")
	}

	var err error
	if endpoint != "" {
		k.endpoint, err = url.Parse(endpoint)
		if err != nil {
			logrus.WithError(err).WithField("endpoint", endpoint).Fatal("Invalid endpoint")
		}
	} else {
		k.dockerCli, err = client.NewEnvClient()
		if err != nil {
			logrus.WithError(err).Fatal("Error creating docker env client")
		}
	}

	k.version, err = k.getKeepalivedVersion()
	if err != nil {
		logrus.WithError(err).Warn("Version detection failed. Assuming it's the latest one.")
	}

	k.initSignals()

	k.initPaths(containerTmpDir)

	return k
}

func (k *KeepalivedContainerCollectorHost) initPaths(containerTmpDir string) {
	k.jsonPath = filepath.Join(containerTmpDir, "keepalived.json")
	k.statsPath = filepath.Join(containerTmpDir, "keepalived.stats")
	k.dataPath = filepath.Join(containerTmpDir, "keepalived.data")
}

// GetKeepalivedVersion returns Keepalived version
func (k *KeepalivedContainerCollectorHost) getKeepalivedVersion() (*version.Version, error) {
	getVersionCmdArgs := []string{"-v"}
	var stdout *bytes.Buffer
	var err error

	if k.containerName != "" {
		stdout, err = k.dockerExecCmd(append([]string{"keepalived"}, getVersionCmdArgs...))
		if err != nil {
			return nil, err
		}
	} else if k.endpoint != nil {
		u := *k.endpoint
		u.Path = filepath.Join(k.endpoint.Path, "version")
		stdout, err = EndpointExec(&u)
		if err != nil {
			return nil, err
		}
	} else {
		cmd := exec.Command("keepalived", getVersionCmdArgs...)
		var stderr *bytes.Buffer
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err := cmd.Run()
		if err != nil {
			logrus.WithFields(logrus.Fields{"stderr": stderr.String(), "stdout": stdout.String()}).WithError(err).Error("Error getting keepalived version")
			return nil, errors.New("Error getting keepalived version")
		}
		stdout = stderr
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

// SigNum returns signal number for given signal name
func (k *KeepalivedContainerCollectorHost) sigNum(sigString string) syscall.Signal {
	if !utils.HasSigNumSupport(k.version) {
		return utils.GetDefaultSignal(sigString)
	}

	sigNumCmdArgs := []string{"--signum", sigString}
	var stdout *bytes.Buffer
	var err error

	if k.containerName != "" {
		stdout, err = k.dockerExecCmd(append([]string{"keepalived"}, sigNumCmdArgs...))
		if err != nil {
			logrus.WithFields(logrus.Fields{"signal": sigString, "container": k.containerName}).WithError(err).Fatal("Error getting signum")
		}
	} else if k.endpoint != nil {
		u := *k.endpoint
		u.Path = filepath.Join(u.Path, "signal/num")
		queryString := u.Query()
		queryString.Set("signal", sigString)
		u.RawQuery = queryString.Encode()
		stdout, err = EndpointExec(&u)
		if err != nil {
			logrus.WithFields(logrus.Fields{"endpoint": k.endpoint, "container": k.containerName}).WithError(err).Fatal("Error getting signum")
		}
	} else {
		cmd := exec.Command("keepalived", sigNumCmdArgs...)
		var stderr bytes.Buffer
		cmd.Stdout = stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err != nil {
			logrus.WithFields(logrus.Fields{"signal": sigString, "stderr": stderr.String()}).WithError(err).Fatal("Error getting signum")
		}
	}

	reg := regexp.MustCompile("[^0-9]+")
	strSigNum := reg.ReplaceAllString(stdout.String(), "")
	signum, err := strconv.ParseInt(strSigNum, 10, 32)
	if err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "signum": stdout.String()}).WithError(err).Fatal("Error parsing signum result")
	}

	return syscall.Signal(signum)
}

// Signal sends signal to Keepalived process
func (k *KeepalivedContainerCollectorHost) signal(signal syscall.Signal) error {
	if k.containerName != "" {
		err := k.dockerCli.ContainerKill(context.Background(), k.containerName, strconv.Itoa(int(signal)))
		if err != nil {
			logrus.WithError(err).WithField("signal", int(signal)).Error("Failed to send signal")
			return err
		}
	} else if k.endpoint != nil {
		u := *k.endpoint
		u.Path = filepath.Join(u.Path, "signal")
		queryString := u.Query()
		queryString.Set("signal", strconv.Itoa(int(signal)))
		u.RawQuery = queryString.Encode()
		_, err := EndpointExec(&u)
		return err
	}

	// Wait 10ms for Keepalived to create its files
	time.Sleep(10 * time.Millisecond)
	return nil
}

// JSONVrrps send SIGJSON and parse the data to the list of collector.VRRP struct
func (k *KeepalivedContainerCollectorHost) JSONVrrps() ([]collector.VRRP, error) {
	err := k.signal(k.SIGJSON)
	if err != nil {
		logrus.WithError(err).Error("Failed to send JSON signal to keepalived")
		return nil, err
	}

	f, err := os.Open(k.jsonPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.jsonPath).Error("Failed to open keepalived.json")
		return nil, err
	}
	defer f.Close()

	return collector.ParseJSON(f)
}

// StatsVrrps send SIGSTATS and parse the stats
func (k *KeepalivedContainerCollectorHost) StatsVrrps() (map[string]*collector.VRRPStats, error) {
	err := k.signal(k.SIGSTATS)
	if err != nil {
		logrus.WithError(err).Error("Failed to send STATS signal to keepalived")
		return nil, err
	}

	f, err := os.Open(k.statsPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.statsPath).Error("Failed to open keepalived.stats")
		return nil, err
	}
	defer f.Close()

	return collector.ParseStats(f)
}

// DataVrrps send SIGDATA ans parse the data
func (k *KeepalivedContainerCollectorHost) DataVrrps() (map[string]*collector.VRRPData, error) {
	err := k.signal(k.SIGDATA)
	if err != nil {
		logrus.WithError(err).Error("Failed to send DATA signal to keepalived")
		return nil, err
	}

	f, err := os.Open(k.dataPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.dataPath).Error("Failed to open keepalived.data")
		return nil, err
	}
	defer f.Close()

	return collector.ParseVRRPData(f)
}

// ScriptVrrps parse the script data from keepalived.data
func (k *KeepalivedContainerCollectorHost) ScriptVrrps() ([]collector.VRRPScript, error) {
	f, err := os.Open(k.dataPath)
	if err != nil {
		logrus.WithError(err).WithField("path", k.dataPath).Error("Failed to open keepalived.data")
		return nil, err
	}
	defer f.Close()

	return collector.ParseVRRPScript(f), nil
}

// HasVRRPScriptStateSupport check if Keepalived version supports VRRP Script State in output
func (k *KeepalivedContainerCollectorHost) HasVRRPScriptStateSupport() bool {
	return utils.HasVRRPScriptStateSupport(k.version)
}
