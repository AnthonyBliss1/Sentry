package deploy

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
)

type TunnelConfig struct {
	Name            string
	ID              string
	Hostname        string
	LocalService    string
	ConfigPath      string
	CredentialsPath string
}

type CloudflaredManager struct {
	Path string
}

type CreateTunnelResult struct {
	ID              string
	CredentialsPath string
}

var uuidPattern = regexp.MustCompile(
	`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`,
)

var credentialsFilePattern = regexp.MustCompile(
	`(?m)([^\s'"]+\.cloudflared[^\s'"]+\.json)`,
)

const cloudflaredLaunchDTemp = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.sentry.cloudflared</string>

	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>tunnel</string>
		<string>--config</string>
		<string>%s</string>
		<string>run</string>
	</array>

	<key>StandardOutPath</key>
	<string>/Library/Logs/com.sentry.cloudflared.out.log</string>

	<key>StandardErrorPath</key>
	<string>/Library/Logs/com.sentry.cloudflared.err.log</string>

	<key>RunAtLoad</key>
	<true/>

	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`

func RunCloudflaredDeploy() error {
	scanner := bufio.NewScanner(os.Stdin)

	utils.Blue.Println("\n> Do you want to configure a Cloudflare tunnel for Sentry Hub? (y/n)")
	utils.Blue.Print("> ")
	if ok := scanConfirm(scanner); !ok {
		return nil
	}

	// need to collect domain (hostname)
	utils.Green.Println("\n[ Starting cloudflared deployment... ]")

	utils.Blue.Println("> Please enter the hostname")
	utils.Blue.Print("> ")

	hostname, err := scanHostname(scanner)
	if err != nil {
		return fmt.Errorf("failed to read hostname: %q", err)
	}

	if err := confirmHostname(&hostname, scanner); err != nil {
		return fmt.Errorf("failed to confirm hostname: %q", err)
	}

	// use default config dir
	configPath, err := defaultSentryConfigPath()
	if err != nil {
		return err
	}

	cfg := TunnelConfig{
		Name:         "sentry-hub-tunnel",
		Hostname:     hostname,
		LocalService: "http://0.0.0.0:8000",
		ConfigPath:   configPath,
	}

	manager := NewCloudflaredManager()
	ctx := context.Background()

	if err := manager.Login(ctx); err != nil {
		return err
	}

	results, err := manager.CreateTunnel(ctx, cfg.Name)
	if err != nil {
		return err
	}

	// store the parsed credentials path and id
	cfg.CredentialsPath = results.CredentialsPath
	cfg.ID = results.ID

	if err := manager.RouteDNS(ctx, cfg.Name, cfg.Hostname); err != nil {
		return err
	}

	if err := manager.WriteConfig(cfg); err != nil {
		return err
	}

	if err := manager.InstallService(ctx, cfg.ConfigPath); err != nil {
		return err
	}

	if err := manager.EnableService(ctx); err != nil {
		return err
	}

	if err := manager.StartService(ctx); err != nil {
		return err
	}

	utils.Green.Println("[ Cloudflare tunnel successfully deployed! ]")

	return nil
}

func NewCloudflaredManager() *CloudflaredManager {
	return &CloudflaredManager{Path: "cloudflared"}
}

func (c *CloudflaredManager) Login(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, c.Path, "tunnel", "login")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (c *CloudflaredManager) CreateTunnel(ctx context.Context, name string) (*CreateTunnelResult, error) {
	cmd := exec.CommandContext(ctx, c.Path, "tunnel", "create", name)

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to create cloudflared tunnel: %q", err)
	}

	results, err := ParseCreateTunnelOutput(output.String())
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (c *CloudflaredManager) RouteDNS(ctx context.Context, tunnelName string, hostname string) error {
	cmd := exec.CommandContext(ctx, c.Path, "tunnel", "route", "dns", tunnelName, hostname)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (c *CloudflaredManager) WriteConfig(cfg TunnelConfig) error {
	if cfg.ID == "" {
		return errors.New("tunnel id required")
	}

	if cfg.Name == "" {
		return errors.New("tunnel name is required")
	}

	if cfg.Hostname == "" {
		return errors.New("hostname is required")
	}

	if cfg.LocalService == "" {
		return errors.New("local service is required")
	}

	if cfg.ConfigPath == "" {
		return errors.New("config path is required")
	}

	if cfg.CredentialsPath == "" {
		return errors.New("credentials path is required")
	}

	contents := fmt.Sprintf(
		`tunnel: %s
credentials-file: %s

ingress:
  - hostname: %s
    service: %s
  - service: http_status:404
`,
		cfg.ID,
		cfg.CredentialsPath,
		cfg.Hostname,
		cfg.LocalService,
	)

	if err := os.MkdirAll(filepath.Dir(cfg.ConfigPath), 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %q", err)
	}

	if err := os.WriteFile(cfg.ConfigPath, []byte(contents), 0600); err != nil {
		return fmt.Errorf("failed to write cloudflared config: %q", err)
	}

	return nil
}

func (c *CloudflaredManager) InstallService(ctx context.Context, configPath string) error {
	switch UserOS {
	case "linux":
		cmd := exec.CommandContext(ctx, "sudo", c.Path, "--config", configPath, "service", "install")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	case "darwin":
		cloudflaredPath, err := exec.LookPath(c.Path)
		if err != nil {
			return fmt.Errorf("failed to find cloudflared path: %w", err)
		}

		plistText := fmt.Sprintf(cloudflaredLaunchDTemp, cloudflaredPath, configPath)

		tmpFile, err := os.CreateTemp("", "com.sentry.cloudflared.*.plist")
		if err != nil {
			return err
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write([]byte(plistText)); err != nil {
			tmpFile.Close()
			return err
		}

		if err := tmpFile.Close(); err != nil {
			return err
		}

		servicePath := "/Library/LaunchDaemons/com.sentry.cloudflared.plist"

		_ = exec.CommandContext(ctx, "sudo", "launchctl", "bootout", "system", servicePath).Run()

		if out, err := exec.CommandContext(ctx, "sudo", "cp", tmpFile.Name(), servicePath).CombinedOutput(); err != nil {
			return fmt.Errorf("copy cloudflared plist failed: %w (%s)", err, string(out))
		}

		if out, err := exec.CommandContext(ctx, "sudo", "chmod", "644", servicePath).CombinedOutput(); err != nil {
			return fmt.Errorf("chmod cloudflared plist failed: %w (%s)", err, string(out))
		}

		if out, err := exec.CommandContext(ctx, "sudo", "chown", "root:wheel", servicePath).CombinedOutput(); err != nil {
			return fmt.Errorf("chown cloudflared plist failed: %w (%s)", err, string(out))
		}

		if out, err := exec.CommandContext(ctx, "sudo", "launchctl", "bootstrap", "system", servicePath).CombinedOutput(); err != nil {
			return fmt.Errorf("bootstrap cloudflared plist failed: %w (%s)", err, string(out))
		}

		return nil

	default:
		return fmt.Errorf("unsupported OS for cloudflared service install: %s", UserOS)
	}
}

func (c *CloudflaredManager) StartService(ctx context.Context) error {
	switch UserOS {
	case "linux":
		cmd := exec.CommandContext(ctx, "sudo", "systemctl", "restart", "cloudflared")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	case "darwin":
		cmd := exec.CommandContext(ctx, "sudo", "launchctl", "kickstart", "-k", "system/com.sentry.cloudflared")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	default:
		return fmt.Errorf("unsupported OS for starting cloudflared service: %s", UserOS)
	}
}

func (c *CloudflaredManager) EnableService(ctx context.Context) error {
	switch UserOS {
	case "linux":
		cmd := exec.CommandContext(ctx, "sudo", "systemctl", "enable", "cloudflared")
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()

	case "darwin":
		return nil

	default:
		return fmt.Errorf("unsupported OS for enabling cloudflared service: %s", UserOS)
	}
}

func ParseCreateTunnelOutput(output string) (*CreateTunnelResult, error) {
	id := uuidPattern.FindString(output)

	if id == "" {
		return nil, errors.New("failed to find tunnel UUID in output")
	}

	var credentialsPath string

	if match := credentialsFilePattern.FindString(output); match != "" {
		credentialsPath = strings.Trim(match, `"'`)
	} else {
		return nil, errors.New("failed to find tunnel credentials path in output")
	}

	return &CreateTunnelResult{ID: id, CredentialsPath: credentialsPath}, nil
}

func defaultSentryConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %q", err)
	}

	return filepath.Join(configDir, "Sentry", "cloudflared", "config.yml"), nil
}

func confirmHostname(hostname *string, scanner *bufio.Scanner) error {
	var ok bool

	fmt.Println()

	for !ok {
		switch *hostname {
		case "":
			utils.Blue.Println("> Please enter the hostname below: ")

			utils.Blue.Print("> ")
			u, err := scanHostname(scanner)
			if err != nil {
				return err
			}

			*hostname = u

		default:
			utils.Blue.Println("> Please confirm the hostname")
			utils.Blue.Printf("> Is %s correct? (y/n)\n", *hostname)

			utils.Blue.Print("> ")
			ok = scanConfirm(scanner)

			if !ok {
				*hostname = ""
			}
		}
	}

	fmt.Println()

	return nil
}

func scanHostname(scanner *bufio.Scanner) (u string, err error) {
	if scanner.Scan() {
		u = scanner.Text()
		u = strings.TrimSpace(u)
	}

	return u, nil
}
