package container

import (
	"bytes"
	"context"
	"io"
	"log/slog"

	"github.com/moby/moby/client"
)

func (k *KeepalivedContainerCollectorHost) dockerExecCmd(cmd []string) (*bytes.Buffer, error) {
	rst, err := k.dockerCli.ExecCreate(context.Background(), k.containerName, client.ExecCreateOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		slog.Error("Error creating exec container", "CMD", cmd, "error", err)

		return nil, err
	}

	response, err := k.dockerCli.ExecAttach(context.Background(), rst.ID, client.ExecAttachOptions{})
	if err != nil {
		slog.Error("Error attaching a connection to an exec process", "CMD", cmd, "error", err)

		return nil, err
	}
	defer response.Close()

	data, err := io.ReadAll(response.Reader)
	if err != nil {
		slog.Error("Error reading response from docker command",
			"error", err,
			"CMD", cmd,
		)

		return nil, err
	}

	return bytes.NewBuffer(data), nil
}
