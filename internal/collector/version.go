package collector

import (
	"errors"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

func (k *KeepalivedCollector) getKeepalivedVersion() (*version.Version, error) {
	output, err := k.collector.GetKeepalivedVersion()
	if err != nil {
		return nil, err
	}

	// version is always at first line
	firstLine, err := output.ReadString('\n')
	if err != nil {
		logrus.WithField("output", output.String()).WithError(err).Error("Failed to parse keepalived version output")
		return nil, errors.New("Failed to parse keepalived version output")
	}

	args := strings.Split(firstLine, " ")
	if len(args) < 2 {
		logrus.WithField("firstLine", firstLine).Error("Unknown keepalived version format")
		return nil, errors.New("Unknown keepalived version format")
	}

	keepalivedVersion, err := version.NewVersion(args[1][1:])
	if err != nil {
		logrus.WithField("version", args[1][1:]).WithError(err).Error("Failed to parse keepalived version")
		return nil, errors.New("Failed to parse keepalived version")
	}

	return keepalivedVersion, nil
}
