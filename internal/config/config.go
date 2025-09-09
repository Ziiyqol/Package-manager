package config

import (
	"fmt"
	"os"
)

// Config содержит все настройки для приложения.
type Config struct {
	SSHUser string
	SSHHost string
	SSHPort int
	SSHKey  string
}

// LoadConfig считывает конфигурацию из переменных окружения.
func LoadConfig() (*Config, error) {
	sshUser := os.Getenv("PM_SSH_USER")
	if sshUser == "" {
		return nil, fmt.Errorf("переменная окружения PM_SSH_USER не установлена")
	}

	sshHost := os.Getenv("PM_SSH_HOST")
	if sshHost == "" {
		return nil, fmt.Errorf("переменная окружения PM_SSH_HOST не установлена")
	}

	sshPortStr := os.Getenv("PM_SSH_PORT")
	if sshPortStr == "" {
		sshPortStr = "22"
	}
	sshPort, err := atoi(sshPortStr)
	if err != nil {
		return nil, fmt.Errorf("некорректное значение для PM_SSH_PORT: %v", err)
	}

	sshKey := os.Getenv("PM_SSH_KEY")
	if sshKey == "" {
		return nil, fmt.Errorf("переменная окружения PM_SSH_KEY не установлена")
	}

	return &Config{
		SSHUser: sshUser,
		SSHHost: sshHost,
		SSHPort: sshPort,
		SSHKey:  sshKey,
	}, nil
}

// atoi для преобразования строки в int, чтобы всё выглядело чисто
func atoi(s string) (int, error) {
	var n int
	_, err := fmt.Sscan(s, &n)
	return n, err
}
