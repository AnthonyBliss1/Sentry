package deploy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	utils "github.com/anthonybliss1/Sentry/Hub/Utils"
)

var UserOS string

const (
	systemDTemp = `[Unit]
Description=sentry-hub
After=network.target

[Service]
User=%s
WorkingDirectory=%s
ExecStart=%s
StandardOutput=append:%s
StandardError=append:%s
Restart=on-failure

[Install]
WantedBy=multi-user.target
`

	launchDTemp = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.sentry.hub</string>

	<key>UserName</key>
	<string>%s</string>

	<key>WorkingDirectory</key>
	<string>%s</string>

	<key>EnvironmentVariables</key>
	<dict>
		<key>HOME</key>
		<string>%s</string>

		<key>PATH</key>
		<string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
	</dict>

	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>

	<key>StandardOutPath</key>
	<string>%s</string>

	<key>StandardErrorPath</key>
	<string>%s</string>

	<key>RunAtLoad</key>
	<true/>

	<key>KeepAlive</key>
	<true/>
</dict>
</plist>
`
)

func DeployHub() error {
	var err error

	utils.Green.Println("[ Deploying Hub Server... ]")

	UserOS, err = checkOS()
	if err != nil {
		return err
	}

	switch UserOS {
	case "linux":
		if err := createSystemdService(); err != nil {
			return err
		}

	case "darwin":
		if err := createLaunchdService(); err != nil {
			return err
		}
	}

	return nil
}

func createSystemdService() error {
	bPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find binary path: %w", err)
	}

	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to find current user: %w", err)
	}

	user := u.Username
	if user == "root" {
		if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
			user = sudoUser
		}
	}

	if err := confirmUser(&user); err != nil {
		return fmt.Errorf("failed to confirm user: %w", err)
	}

	bDir := filepath.Dir(bPath)

	utils.Green.Println("[ Binary path identified ... ]")

	logPath := "/var/log/sentry/sentry-hub.log"

	// make sure log directory exists
	if out, err := exec.Command("sudo", "mkdir", "-p", "/var/log/sentry").CombinedOutput(); err != nil {
		return fmt.Errorf("mkdir log dir failed: %q (%s)", err, string(out))
	}

	unitText := fmt.Sprintf(systemDTemp, user, bDir, bPath, logPath, logPath)

	utils.Green.Println("[ Creating unit file ... ]")

	servicePath := "/etc/systemd/system/sentry-hub.service"
	tmpServicePath := "/tmp/sentry-hub.service"

	if err := os.WriteFile(tmpServicePath, []byte(unitText), 0o644); err != nil {
		return err
	}

	if out, err := exec.Command("sudo", "cp", tmpServicePath, servicePath).CombinedOutput(); err != nil {
		return fmt.Errorf("copy service file failed: %q (%s)", err, string(out))
	}

	if out, err := exec.Command("sudo", "chmod", "644", servicePath).CombinedOutput(); err != nil {
		return fmt.Errorf("chmod service file failed: %q (%s)", err, string(out))
	}

	utils.Green.Println("[ Reloading systemctl daemon ... ]")

	if out, err := exec.Command("sudo", "systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload failed: %q (%s)", err, string(out))
	}

	utils.Green.Println("[ Enabling service ... ]")

	if out, err := exec.Command("sudo", "systemctl", "enable", "sentry-hub.service").CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "is enabled") {
			return fmt.Errorf("enable failed: %q (%s)", err, string(out))
		}
	}

	utils.Green.Println("[ Restarting service ... ]")

	if out, err := exec.Command("sudo", "systemctl", "restart", "sentry-hub.service").CombinedOutput(); err != nil {
		return fmt.Errorf("restart failed: %q (%s)", err, string(out))
	}

	return nil
}

func createLaunchdService() error {
	bPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find binary path: %w", err)
	}

	u, err := user.Current()
	if err != nil {
		return fmt.Errorf("failed to find current user: %w", err)
	}

	username := u.Username
	if username == "root" {
		if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
			username = sudoUser
		}
	}

	if err := confirmUser(&username); err != nil {
		return fmt.Errorf("failed to confirm user: %w", err)
	}

	bDir := filepath.Dir(bPath)

	utils.Green.Println("[ Binary path identified ... ]")

	logPath := "/var/log/sentry/sentry-hub.log"

	// make sure log directory exists
	if out, err := exec.Command("sudo", "mkdir", "-p", filepath.Dir(logPath)).CombinedOutput(); err != nil {
		return fmt.Errorf("mkdir log dir failed: %q (%s)", err, string(out))
	}

	if out, err := exec.Command("sudo", "touch", logPath).CombinedOutput(); err != nil {
		return fmt.Errorf("touch log failed: %q (%s)", err, string(out))
	}

	if out, err := exec.Command("sudo", "chmod", "644", logPath).CombinedOutput(); err != nil {
		return fmt.Errorf("chmod log failed: %q (%s)", err, string(out))
	}

	if out, err := exec.Command("sudo", "chown", username, logPath).CombinedOutput(); err != nil {
		return fmt.Errorf("chown log failed: %q (%s)", err, string(out))
	}

	servicePath := "/Library/LaunchDaemons/com.sentry.hub.plist"

	utils.Green.Println("[ Unloading existing launchd service ... ]")
	exec.Command("sudo", "launchctl", "bootout", "system", servicePath).Run()

	utils.Green.Println("[ Creating launchd plist ... ]")

	targetUser, err := user.Lookup(username)
	if err != nil {
		return err
	}

	homeDir := targetUser.HomeDir

	plistText := fmt.Sprintf(launchDTemp, username, bDir, homeDir, bPath, logPath, logPath)
	tmpPlistPath := "/tmp/com.sentry.hub.plist"

	if err := os.WriteFile(tmpPlistPath, []byte(plistText), 0o644); err != nil {
		return err
	}

	if out, err := exec.Command("sudo", "cp", tmpPlistPath, servicePath).CombinedOutput(); err != nil {
		return fmt.Errorf("copy plist failed: %q (%s)", err, string(out))
	}

	if out, err := exec.Command("sudo", "chmod", "644", servicePath).CombinedOutput(); err != nil {
		return fmt.Errorf("chmod plist failed: %q (%s)", err, string(out))
	}

	if out, err := exec.Command("sudo", "chown", "root:wheel", servicePath).CombinedOutput(); err != nil {
		return fmt.Errorf("chown failed: %q (%s)", err, string(out))
	}

	utils.Green.Println("[ Bootstrapping launchd service ... ]")

	if out, err := exec.Command("sudo", "launchctl", "bootstrap", "system", servicePath).CombinedOutput(); err != nil {
		return fmt.Errorf("bootstrap failed: %q (%s)", err, string(out))
	}

	utils.Green.Println("[ Restarting launchd service ... ]")

	if out, err := exec.Command("sudo", "launchctl", "kickstart", "-k", "system/com.sentry.hub").CombinedOutput(); err != nil {
		return fmt.Errorf("kickstart failed: %q (%s)", err, string(out))
	}

	return nil
}

func checkOS() (o string, err error) {
	o = runtime.GOOS

	// make sure os is supported
	if o != "darwin" && o != "linux" {
		return "", fmt.Errorf("%s not supported", o)
	}

	return o, nil
}

func confirmUser(user *string) error {
	scanner := bufio.NewScanner(os.Stdin)

	var ok bool

	fmt.Println()

	for !ok {
		switch *user {
		case "":
			utils.Blue.Println("> Please enter the user below: ")

			utils.Blue.Print("> ")
			u, err := scanUser(scanner)
			if err != nil {
				return err
			}

			*user = u

		default:
			utils.Blue.Println("> Please confirm the user")
			utils.Blue.Printf("> Is %s correct? (y/n)\n", *user)

			utils.Blue.Print("> ")
			ok = scanConfirm(scanner)

			if !ok {
				*user = ""
			}
		}
	}

	fmt.Println()

	return nil
}

func scanUser(scanner *bufio.Scanner) (u string, err error) {
	if scanner.Scan() {
		u = scanner.Text()
		u = strings.TrimSpace(u)

		if _, err := user.Lookup(u); err != nil {
			return "", err
		}
	}

	return u, nil
}

func scanConfirm(scanner *bufio.Scanner) bool {
	var input string

	if scanner.Scan() {
		input = scanner.Text()
		input = strings.ToLower(input)
	}

	return input == "y"
}
