// Copyright 2024 Daytona Platforms Inc.
// SPDX-License-Identifier: Apache-2.0

package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/daytonaio/daytona/cmd/daytona/config"
	"github.com/daytonaio/daytona/pkg/models"
	"github.com/daytonaio/daytona/pkg/ports"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"
	log "github.com/sirupsen/logrus"
)

func (d *DockerClient) initWorkspaceContainer(target *models.Target, logWriter io.Writer) error {
	ctx := context.Background()
	mounts := []mount.Mount{}

	configPath, err := config.GetConfigDir()
	if err != nil {
		return fmt.Errorf("error getting config dir: %w", err)
	}

	winStorage := filepath.Join(configPath, "server", "local-runner", "providers", "mac-provider", "mac")
	err = os.MkdirAll(winStorage, 0755)
	if err != nil {
		return err
	}

	mounts = append(mounts, mount.Mount{
		Type:   mount.TypeBind,
		Source: winStorage,
		Target: "/storage",
	})

	var availablePort *uint16
	portBindings := make(map[nat.Port][]nat.PortBinding)
	portBindings["22/tcp"] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: "10022",
		},
	}
	portBindings["2222/tcp"] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: "2222",
		},
	}
	portBindings["8006/tcp"] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: "8006",
		},
	}

	if d.IsLocalWindowsTarget(target.TargetConfig.ProviderInfo.Name, target.TargetConfig.Options, target.TargetConfig.ProviderInfo.RunnerId) {
		p, err := ports.GetAvailableEphemeralPort()
		if err != nil {
			log.Error(err)
		} else {
			availablePort = &p
			portBindings["2280/tcp"] = []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: fmt.Sprintf("%d", *availablePort),
				},
			}
		}
	}

	c, err := d.apiClient.ContainerCreate(ctx, GetContainerCreateConfig(target, availablePort), &container.HostConfig{
		Privileged: true,
		Mounts:     mounts,
		ExtraHosts: []string{
			"host.docker.internal:host-gateway",
		},
		PortBindings: portBindings,
		Resources: container.Resources{
			Devices: []container.DeviceMapping{
				{
					PathOnHost:      "/dev/kvm",
					PathInContainer: "/dev/kvm",
				},
				{
					PathOnHost:      "/dev/net/tun",
					PathInContainer: "/dev/net/tun",
				},
			},
		},
		CapAdd: []string{
			"NET_ADMIN",
			"SYS_ADMIN",
		},
	}, nil, nil, d.GetTargetContainerName(target))
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	err = d.apiClient.ContainerStart(ctx, c.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	for {
		c, err := d.apiClient.ContainerInspect(ctx, c.ID)
		if err != nil {
			return fmt.Errorf("failed to inspect container when creating project: %w", err)
		}

		if c.State.Running {
			break
		}

		time.Sleep(1 * time.Second)
	}

	logWriter.Write([]byte("Installing Windows.....\n"))

	d.OpenWebUI(d.targetOptions.RemoteHostname, logWriter)

	err = d.WaitForWindowsBoot(c.ID, d.targetOptions.RemoteHostname)
	if err != nil {
		return fmt.Errorf("failed to wait for Windows to boot: %w", err)
	}

	sshClient, err := d.GetSshClient(d.targetOptions.RemoteHostname)
	if err != nil {
		return fmt.Errorf("failed to get SSH client: %w", err)
	}

	for key, env := range target.EnvVars {
		err = d.ExecuteCommand(fmt.Sprintf("setx %s \"%s\"", key, env), nil, sshClient)
		if err != nil {
			logWriter.Write([]byte(fmt.Sprintf("failed to set env variable %s to %s: %s\n", key, env, err.Error())))
		}
	}

	err = d.ExecuteCommand(fmt.Sprintf("setx HOME \"%s\"", "C:\\Users\\daytona"), nil, sshClient)
	if err != nil {
		logWriter.Write([]byte("failed to set env variable DAYTONA_AGENT_LOG_FILE_PATH to C:\\Users\\daytona\\.daytona-agent.log\n"))
	}

	return nil
}

func GetContainerCreateConfig(target *models.Target, toolboxApiHostPort *uint16) *container.Config {
	envVars := []string{
		fmt.Sprintf("ARGUMENTS=%s", "-device e1000,netdev=net0  -netdev user,id=net0,hostfwd=tcp::22-:22,hostfwd=tcp::2222-:2222 "),
		fmt.Sprintf("RAM_SIZE=%s", "2G"),
	}
	for key, value := range target.EnvVars {
		envVars = append(envVars, fmt.Sprintf("%s=%s", key, value))
	}

	labels := map[string]string{
		"daytona.target.id":   target.Id,
		"daytona.target.name": target.Name + "-daytona-mac",
	}

	if toolboxApiHostPort != nil {
		labels["daytona.toolbox.api.hostPort"] = fmt.Sprintf("%d", *toolboxApiHostPort)
	}

	exposedPorts := nat.PortSet{}
	if toolboxApiHostPort != nil {
		exposedPorts["2280/tcp"] = struct{}{}
	}

	return &container.Config{
		Hostname:   target.Name,
		WorkingDir: fmt.Sprintf("/data/%s", target.Name),
		Image:      "dockurr/mac:latest",
		Labels:     labels,
		User:       "root",
		Entrypoint: []string{
			"/usr/bin/tini",
			"-s",
			"/run/entry.sh",
		},
		Env:          envVars,
		AttachStdout: true,
		AttachStderr: true,
		ExposedPorts: exposedPorts,
		StopTimeout:  &[]int{120}[0],
	}
}
