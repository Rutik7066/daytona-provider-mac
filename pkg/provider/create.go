package provider

import (
	"errors"
	"io"

	log_writers "github.com/Rutik7066/daytona-provider-macos/internal/log"

	"github.com/Rutik7066/daytona-provider-macos/pkg/docker"
	"github.com/Rutik7066/daytona-provider-macos/pkg/types"
	"github.com/daytonaio/daytona/pkg/logs"
	"github.com/daytonaio/daytona/pkg/provider"
	provider_util "github.com/daytonaio/daytona/pkg/provider/util"
	"github.com/daytonaio/daytona/pkg/ssh"
)

func (p MacProvider) CreateTarget(targetReq *provider.TargetRequest) (*provider_util.Empty, error) {
	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.TargetLogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(logs.LoggerFactoryConfig{
			LogsDir:     *p.TargetLogsDir,
			ApiUrl:      p.ApiUrl,
			ApiKey:      p.ApiKey,
			ApiBasePath: &logs.ApiBasePathTarget,
		})
		targetLogWriter, err := loggerFactory.CreateLogger(targetReq.Target.Id, targetReq.Target.Name, logs.LogSourceProvider)
		if err != nil {
			return new(provider_util.Empty), err
		}
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, targetLogWriter)
		defer targetLogWriter.Close()
	}

	dockerClient, err := p.getClient(targetReq.Target.TargetConfig.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	targetDir, err := p.getTargetDir(targetReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	sshClient, err := p.getSshClient(targetReq.Target.TargetConfig.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}
	if sshClient != nil {
		defer sshClient.Close()
	}

	return new(provider_util.Empty), dockerClient.CreateTarget(targetReq.Target, targetDir, logWriter, sshClient)
}

func (p MacProvider) CreateWorkspace(workspaceReq *provider.WorkspaceRequest) (*provider_util.Empty, error) {
	if p.DaytonaDownloadUrl == nil {
		return new(provider_util.Empty), errors.New("ServerDownloadUrl not set. Did you forget to call Initialize?")
	}

	logWriter := io.MultiWriter(&log_writers.InfoLogWriter{})
	if p.WorkspaceLogsDir != nil {
		loggerFactory := logs.NewLoggerFactory(logs.LoggerFactoryConfig{
			LogsDir:     *p.WorkspaceLogsDir,
			ApiUrl:      p.ApiUrl,
			ApiKey:      p.ApiKey,
			ApiBasePath: &logs.ApiBasePathWorkspace,
		})
		workspaceLogWriter, err := loggerFactory.CreateLogger(workspaceReq.Workspace.Id, workspaceReq.Workspace.Name, logs.LogSourceProvider)
		if err != nil {
			return new(provider_util.Empty), err
		}
		logWriter = io.MultiWriter(&log_writers.InfoLogWriter{}, workspaceLogWriter)
		defer workspaceLogWriter.Close()
	}

	dockerClient, err := p.getClient(workspaceReq.Workspace.Target.TargetConfig.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	workspaceDir, err := p.getWorkspaceDir(workspaceReq)
	if err != nil {
		return new(provider_util.Empty), err
	}

	_, isLocal, err := types.ParseTargetConfigOptions(workspaceReq.Workspace.Target.TargetConfig.Options)
	if err != nil {
		return new(provider_util.Empty), err
	}

	var sshClient *ssh.Client
	if !isLocal {
		sshClient, err = p.getSshClient(workspaceReq.Workspace.Target.TargetConfig.Options)
		if err != nil {
			return new(provider_util.Empty), err
		}
		if sshClient != nil {
			defer sshClient.Close()
		}
	}

	return new(provider_util.Empty), dockerClient.CreateWorkspace(&docker.CreateWorkspaceOptions{
		Workspace:           workspaceReq.Workspace,
		WorkspaceDir:        workspaceDir,
		ContainerRegistries: workspaceReq.ContainerRegistries,
		BuilderImage:        workspaceReq.BuilderImage,
		LogWriter:           logWriter,
		Gpc:                 workspaceReq.GitProviderConfig,
		SshClient:           sshClient,
	})
}
