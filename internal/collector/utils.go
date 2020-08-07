package collector

import (
	"bytes"
	"os/exec"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

func keepalivedVersion() *version.Version {
	cmd := exec.Command("bash", "-c", "keepalived -v")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logrus.WithFields(logrus.Fields{"stderr": stderr.String(), "stdout": stdout.String()}).WithError(err).Fatal("Error getting keepalived version")
	}

	// version is always at first line
	firstLine, err := stderr.ReadString('\n')
	if err != nil {
		logrus.WithField("output", stderr.String()).WithError(err).Fatal("Failed to parse keepalived version output")
	}

	args := strings.Split(firstLine, " ")
	if len(args) < 2 {
		logrus.WithField("firstLine", firstLine).Fatal("Unknown keepalived version format")
	}

	v, err := version.NewVersion(args[1][1:])
	if err != nil {
		logrus.WithField("version", args[1][1:]).WithError(err).Fatal("Failed to parse keepalived version")
	}

	return v
}
