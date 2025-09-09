package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	SSHUser string
	SSHHost string
	SSHPort int
	SSHKey  string
}

func LoadConfig() (*Config, error) {
	sshUser := os.Getenv("PM_SSH_USER")
	if sshUser == "" {
		return nil, fmt.Errorf("environment variable PM_SSH_USER is not set")
	}

	sshHost := os.Getenv("PM_SSH_HOST")
	if sshHost == "" {
		return nil, fmt.Errorf("environment variable PM_SSH_HOST is not set")
	}

	sshPortStr := os.Getenv("PM_SSH_PORT")
	if sshPortStr == "" {
		sshPortStr = "22"
	}
	sshPort, err := strconv.Atoi(sshPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid value for PM_SSH_PORT: %w", err)
	}

	if sshPort < 1 || sshPort > 65535 {
		return nil, fmt.Errorf("PM_SSH_PORT must be between 1 and 65535")
	}

	sshKey := os.Getenv("PM_SSH_KEY")
	if sshKey == "" {
		return nil, fmt.Errorf("environment variable PM_SSH_KEY is not set")
	}

	return &Config{
		SSHUser: sshUser,
		SSHHost: sshHost,
		SSHPort: sshPort,
		SSHKey:  sshKey,
	}, nil
}
