package collector

import (
	"encoding/json"
	"os"
	"syscall"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

var sigNumSupportedVersion = version.Must(version.NewVersion("1.3.8"))

func (k *KeepalivedCollector) isSigNumSupport() bool {
	keepalivedVersion, err := k.getKeepalivedVersion()
	if err != nil {
		// keep backward compatibility and assuming it's the latest one on version detection failure
		return true
	}
	return keepalivedVersion.GreaterThanOrEqual(sigNumSupportedVersion)
}

func (k *KeepalivedCollector) sigNum(sig string) os.Signal {
	if !k.isSigNumSupport() {
		switch sig {
		case "DATA":
			return syscall.SIGUSR1
		case "STATS":
			return syscall.SIGUSR2
		default:
			logrus.WithField("signal", sig).Fatal("Unsupported signal for your keepalived")
		}
	}

	output := k.collector.SigNum(sig)

	var signum int
	err := json.Unmarshal(output.Bytes(), &signum)
	if err != nil {
		logrus.WithFields(logrus.Fields{"signal": sig, "signum": output.String()}).WithError(err).Fatal("Error unmarshalling signum result")
	}

	return syscall.Signal(signum)
}
