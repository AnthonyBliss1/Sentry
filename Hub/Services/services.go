package services

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
)

//go:embed mtx-compose.yml mediamtx.yml Dockerfile.detect obj-detect-compose.yml detect-object.py
var embeddedConfigs embed.FS

type ContainerController struct {
	Containers []*Container
}

type Container struct {
	*Service
	*Composer
}

type Service struct {
	Name       string
	ComposeYML string
	Files      []string
}

type Composer struct {
	Service api.Compose
	Project *types.Project
	Ctx     context.Context
}

func (c *ContainerController) CreateAllComposers() error {
	for _, container := range c.Containers {
		var err error

		container.Composer, err = container.Service.CreateComposer()
		if err != nil {
			return fmt.Errorf("failed to create composer for %s: %q", container.Name)
		}
	}

	return nil
}

func (c *ContainerController) StartAllServices() error {
	for _, container := range c.Containers {
		if err := container.Composer.Service.Up(container.Ctx, container.Project, api.UpOptions{
			Create: api.CreateOptions{
				Build: &api.BuildOptions{
					Progress: "plain",
					Out:      os.Stdout,
				},
			},
		}); err != nil {
			return fmt.Errorf("failed to startup %s: %q", container.Name, err)
		}
	}

	return nil
}

func (c *ContainerController) StopAllServices() error {
	for _, container := range c.Containers {
		if err := container.Composer.Service.Down(container.Ctx, container.Project.Name, api.DownOptions{Images: "all"}); err != nil {
			return fmt.Errorf("failed to stop %s: %q", container.Name, err)
		}
	}

	return nil
}

func (s *Service) CreateComposer() (*Composer, error) {
	ctx := context.Background()

	workDir, err := writeEmbeddedConfigs(s.Files)
	if err != nil {
		return nil, err
	}

	dockerCLI, err := command.NewDockerCli()
	if err != nil {
		return nil, fmt.Errorf("failed to create new docker cli: %w", err)
	}

	if err := dockerCLI.Initialize(&flags.ClientOptions{}); err != nil {
		return nil, fmt.Errorf("failed to initialize dockerCLI: %w", err)
	}

	service, err := compose.NewComposeService(dockerCLI)
	if err != nil {
		return nil, fmt.Errorf("failed to create compose service: %w", err)
	}

	project, err := service.LoadProject(ctx, api.ProjectLoadOptions{
		ConfigPaths: []string{filepath.Join(workDir, s.ComposeYML)},
		ProjectName: s.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load project with yml file: %w", err)
	}

	return &Composer{service, project, ctx}, nil
}

func writeEmbeddedConfigs(files []string) (string, error) {
	workDir, err := os.MkdirTemp("", "sentry-hub-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp working dir: %w", err)
	}

	for _, name := range files {
		data, err := embeddedConfigs.ReadFile(name)
		if err != nil {
			return "", fmt.Errorf("failed to read embedded %s: %w", name, err)
		}

		dst := filepath.Join(workDir, name)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return "", fmt.Errorf("failed to write %s: %w", name, err)
		}
	}

	return workDir, nil
}
