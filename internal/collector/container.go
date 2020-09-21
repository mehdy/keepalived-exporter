package collector

import (
	"bytes"
	"context"
	"io/ioutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
)

func dockerExecCmd(cmd []string, container string) (*bytes.Buffer, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error creating docker client")
		return nil, err
	}

	rst, err := cli.ContainerExecCreate(context.Background(), container, types.ExecConfig{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error creating exec container")
		return nil, err
	}

	response, err := cli.ContainerExecAttach(context.Background(), rst.ID, types.ExecConfig{})
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error attaching a connection to an exec process")
		return nil, err
	}
	defer response.Close()

	data, err := ioutil.ReadAll(response.Reader)
	if err != nil {
		logrus.WithError(err).WithField("CMD", cmd).Error("Error reading response from docker exec command")
		return nil, err
	}

	return bytes.NewBuffer(data), nil
}

func dockerKillContainer(container, signal string) error {
	cli, err := client.NewEnvClient()
	if err != nil {
		logrus.WithError(err).WithField("signal", signal).Error("Error creating docker client")
		return err
	}

	return cli.ContainerKill(context.Background(), container, signal)
}
