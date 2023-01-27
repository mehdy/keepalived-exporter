package utils

import (
	"errors"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

// ParseVersion returns keepalived version from keepalived -v command output.
func ParseVersion(versionOutput string) (*version.Version, error) {
	// version is always at first line
	lines := strings.SplitN(versionOutput, "\n", 2)
	if len(lines) != 2 {
		logrus.WithField("output", versionOutput).Error("Failed to parse keepalived version output")
		return nil, errors.New("failed to parse keepalived version output")
	}
	versionString := lines[0]

	args := strings.Split(versionString, " ")
	if len(args) < 2 {
		logrus.WithField("version", versionString).Error("Unknown keepalived version format")
		return nil, errors.New("unknown keepalived version format")
	}

	version, err := version.NewVersion(args[1][1:])
	if err != nil {
		logrus.WithField("version", args[1][1:]).WithError(err).Error("Failed to parse keepalived version")
		return nil, errors.New("failed to parse keepalived version")
	}

	return version, nil
}
