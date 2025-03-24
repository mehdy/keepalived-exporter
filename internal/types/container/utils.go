package container

import (
	"bytes"
	"context"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/sirupsen/logrus"
)

func (k *KeepalivedContainerCollectorHost) dockerExecCmd(cmd []string) (*bytes.Buffer, error) {
	rst, err := k.dockerCli.ContainerExecCreate(context.Background(), k.containerName, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error creating exec container")

		return nil, err
	}

	response, err := k.dockerCli.ContainerExecAttach(context.Background(), rst.ID, container.ExecStartOptions{})
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error attaching a connection to an exec process")

		return nil, err
	}
	defer response.Close()

	data, err := io.ReadAll(response.Reader)
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error reading response from docker command")

		return nil, err
	}

	return bytes.NewBuffer(data), nil
}
