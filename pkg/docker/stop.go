// Copyright 2024 Daytona Platforms Inc.
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"io"
)

func (d *DockerClient) StopTarget(logWriter io.Writer) error {
	sshClient, err := d.GetSshClient(d.targetOptions.RemoteHostname)
	if err != nil {
		return err
	}

	return d.ExecuteCommand("shutdown /s /t 0", logWriter, sshClient)
}
