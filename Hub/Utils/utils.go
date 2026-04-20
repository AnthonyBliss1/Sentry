package utils

import (
	"context"
	"embed"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/flags"
	"github.com/docker/compose/v5/pkg/api"
	"github.com/docker/compose/v5/pkg/compose"
	"github.com/fatih/color"
)

//go:embed docker-compose.yml mediamtx.yml
var embeddedConfigs embed.FS

var (
	Green = color.New(color.FgGreen) // debug
	Blue  = color.New(color.FgBlue)  // actions
	Red   = color.New(color.FgRed)   // warnings
)

type Composer struct {
	Service api.Compose
	Project *types.Project
	Ctx     context.Context
}

func LANIPv4() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

func writeEmbeddedConfigs() (string, error) {
	workDir, err := os.MkdirTemp("", "sentry-hub-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp working dir: %w", err)
	}

	files := []string{"docker-compose.yml", "mediamtx.yml"}

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

func CreateComposer() (Composer, error) {
	ctx := context.Background()

	workDir, err := writeEmbeddedConfigs()
	if err != nil {
		return Composer{}, err
	}

	dockerCLI, err := command.NewDockerCli()
	if err != nil {
		return Composer{}, fmt.Errorf("failed to create new docker cli: %w", err)
	}

	if err := dockerCLI.Initialize(&flags.ClientOptions{}); err != nil {
		return Composer{}, fmt.Errorf("failed to initialize dockerCLI: %w", err)
	}

	service, err := compose.NewComposeService(dockerCLI)
	if err != nil {
		return Composer{}, fmt.Errorf("failed to create compose service: %w", err)
	}

	project, err := service.LoadProject(ctx, api.ProjectLoadOptions{
		ConfigPaths: []string{filepath.Join(workDir, "docker-compose.yml")},
		ProjectName: "sentry-hub-mediamtx",
	})
	if err != nil {
		return Composer{}, fmt.Errorf("failed to load project with yml file: %w", err)
	}

	return Composer{service, project, ctx}, nil
}
