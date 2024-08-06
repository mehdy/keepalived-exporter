package host

import (
	"bytes"
	"encoding/json"

	"github.com/sirupsen/logrus"
)

func parseSigNum(sigNum bytes.Buffer, sigString string) int64 {
	var signum int64
	if err := json.Unmarshal(sigNum.Bytes(), &signum); err != nil {
		logrus.WithFields(logrus.Fields{"signal": sigString, "signum": sigNum.String()}).
			WithError(err).
			Fatal("Error parsing signum result")
	}

	return signum
}
