package collector

import (
	"bytes"
	"errors"
	"os/exec"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

func getKeepalivedVersion() (*version.Version, error) {
	cmd := exec.Command("bash", "-c", "keepalived -v")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logrus.WithFields(logrus.Fields{"stderr": stderr.String(), "stdout": stdout.String()}).WithError(err).Error("Error getting keepalived version")
		return nil, errors.New("Error getting keepalived version")
	}

	// version is always at first line
	firstLine, err := stderr.ReadString('\n')
	if err != nil {
		logrus.WithField("output", stderr.String()).WithError(err).Error("Failed to parse keepalived version output")
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
