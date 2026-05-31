package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fatih/color"
)

var (
	Hostname string
	Green    = color.New(color.FgGreen) // debug
	Blue     = color.New(color.FgBlue)  // actions
	Red      = color.New(color.FgRed)   // warnings
)

type Alias struct {
	Alias string `json:"alias"`
}

func init() {
	// override the hostname if there's an alias present
	// if the hub server fails for whatever reason, and the stream needs to be restarted,
	// this will need to be executed again (in memory alias store on the hub will be wiped if restarted)
	if err := SetHostname(); err != nil {
		Red.Println(err)
	}
}

func SetHostname() error {
	// first check if there is a stored alias
	alias, err := GetAlias()
	if err != nil {
		return err
	}

	// prioritize the stored alias over the machines hostname
	if alias != "" {
		Hostname = alias
	} else {
		Hostname, err = os.Hostname()
		if err != nil {
			log.Fatalf("failed to collect hostname: %q", err)
		}
	}

	return nil
}

func SetAlias(alias string) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to find user directory: %q", err)
	}

	sentryDir := filepath.Join(configDir, "Sentry")

	if err := os.MkdirAll(sentryDir, 0o755); err != nil {
		return fmt.Errorf("failed to make Sentry config dir: %q", err)
	}

	aliasSt := Alias{alias}

	aliasJSON, err := json.Marshal(aliasSt)
	if err != nil {
		return fmt.Errorf("failed to encode alias: %q", err)
	}

	aliasPath := filepath.Join(sentryDir, "alias.json")

	if err := os.WriteFile(aliasPath, aliasJSON, 0o644); err != nil {
		return fmt.Errorf("failed to write alias.json file: %q", err)
	}

	return nil
}

func GetAlias() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to find user directory: %q", err)
	}

	aliasPath := filepath.Join(configDir, "Sentry", "alias.json")

	aliasJSON, err := os.ReadFile(aliasPath)
	if err != nil {
		return "", fmt.Errorf("failed to read alias.json: %q", err)
	}

	var alias Alias

	if err := json.Unmarshal(aliasJSON, &alias); err != nil {
		return "", fmt.Errorf("failed to decode alias.json: %q", err)
	}

	return alias.Alias, nil
}
