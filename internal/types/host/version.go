package host

import (
	"bytes"
	"errors"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// GetKeepalivedVersion returns Keepalived version
func (khch *KeepalivedHostCollectorHost) GetKeepalivedVersion() (*bytes.Buffer, error) {
	cmd := exec.Command("bash", "-c", "keepalived -v")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		logrus.WithFields(logrus.Fields{"stderr": stderr.String(), "stdout": stdout.String()}).WithError(err).Error("Error getting keepalived version")
		return nil, errors.New("Error getting keepalived version")
	}

	return &stderr, nil
}
