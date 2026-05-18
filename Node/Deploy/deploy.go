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

	utils "github.com/anthonybliss1/Sentry/Node/Utils"
)

const (
	systemDTemp = `[Unit]
Description=sentry-node
After=network.target

[Service]
User=%s
WorkingDirectory=%s
ExecStart=%s
StandardOutput=journal
StandardError=journal
Restart=on-failure

[Install]
WantedBy=multi-user.target
`
)

func DeployNode() error {
	utils.Green.Println("[ Creating Node Service... ]")

	os, err := checkOS()
	if err != nil {
		return err
	}

	switch os {
	case "linux":
		if err := createSystemdService(); err != nil {
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

	unitText := fmt.Sprintf(systemDTemp, user, bDir, bPath)

	utils.Green.Println("[ Creating unit file ... ]")

	servicePath := "/etc/systemd/system/sentry-node.service"
	if err := os.WriteFile(servicePath, []byte(unitText), 0o644); err != nil {
		return err
	}

	utils.Green.Println("[ Reloading systemctl daemon ... ]")

	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("daemon-reload failed: %q (%s)", err, string(out))
	}

	utils.Green.Println("[ Enabling service ... ]")

	if out, err := exec.Command("systemctl", "enable", "sentry-node.service").CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "is enabled") {
			return fmt.Errorf("enable failed: %q (%s)", err, string(out))
		}
	}

	utils.Green.Println("[ Restarting service ... ]")

	if out, err := exec.Command("systemctl", "restart", "sentry-node.service").CombinedOutput(); err != nil {
		return fmt.Errorf("restart failed: %q (%s)", err, string(out))
	}

	return nil
}

func checkOS() (o string, err error) {
	o = runtime.GOOS

	// make sure os is supported
	if o != "linux" {
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
