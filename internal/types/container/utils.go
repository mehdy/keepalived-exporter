package container

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/docker/docker/api/types"
	"github.com/sirupsen/logrus"
)

func (k *KeepalivedContainerCollectorHost) dockerExecCmd(cmd []string) (*bytes.Buffer, error) {
	rst, err := k.dockerCli.ContainerExecCreate(context.Background(), k.containerName, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error creating exec container")
		return nil, err
	}

	response, err := k.dockerCli.ContainerExecAttach(context.Background(), rst.ID, types.ExecConfig{})
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error attaching a connection to an exec process")
		return nil, err
	}
	defer response.Close()

	data, err := ioutil.ReadAll(response.Reader)
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error reading response from docker command")
		return nil, err
	}

	return bytes.NewBuffer(data), nil
}

// EndpointExec execute command with HTTP on Keepalived host
func EndpointExec(u fmt.Stringer) (*bytes.Buffer, error) {
	response, err := http.Get(u.String())
	if err != nil {
		logrus.WithField("url", u).WithError(err).Error("Error sending request to endpoint")
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		logrus.WithField("statuscode", response.StatusCode).Error("Request was not successful")
		return nil, errors.New("Request was not successful")
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		logrus.WithError(err).Error("Error parsing response")
		return nil, err
	}

	return bytes.NewBuffer(body), nil
}
