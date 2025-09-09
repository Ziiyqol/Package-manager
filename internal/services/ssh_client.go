package services

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"package-manager/internal/config"
)

// SSHClient инкапсулирует логику для работы с SSH-соединением.
// Реализует интерфейс SSHClientInterface
type SSHClient struct {
	config *config.Config
	client *ssh.Client
	mu     sync.Mutex
}

// NewSSHClient создает новый экземпляр SSHClient
func NewSSHClient(cfg *config.Config) *SSHClient {
	return &SSHClient{config: cfg}
}

// connect устанавливает/возвращает SSH-соединение
func (c *SSHClient) connect() (*ssh.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		// Проверяем, живое ли соединение
		_, err := c.client.NewSession()
		if err == nil {
			return c.client, nil // Живое - возвращаем
		}
		// Если нет, закрываем и переподключаемся
		c.client.Close()
		c.client = nil
	}

	// Чтение ключа
	key, err := os.ReadFile(c.config.SSHKey)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать SSH-ключ: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("не удалось разобрать SSH-ключ: %w", err)
	}

	// Получаем домашнюю директорию
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("не удалось определить домашнюю директорию: %w", err)
	}
	knownHostsPath := filepath.Join(homeDir, ".ssh", "known_hosts")

	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении файла known_hosts: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.config.SSHUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: hostKeyCallback,
	}

	addr := fmt.Sprintf("%s:%d", c.config.SSHHost, c.config.SSHPort)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ошибка SSH-соединения: %w", err)
	}

	c.client = client
	return client, nil
}

// Close закрывает SSH-клиент (вызывать при shutdown сервиса)
func (c *SSHClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.client != nil {
		err := c.client.Close()
		c.client = nil
		return err
	}
	return nil
}

// UploadFile загружает файл на удаленный сервер, используя SCP для более простой и быстрой реализации
func (c *SSHClient) UploadFile(fileName string, data *bytes.Buffer) error {
	client, err := c.connect()
	if err != nil {
		return err
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("ошибка создания SSH-сессии: %w", err)
	}
	defer session.Close()

	w, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("ошибка получения StdinPipe: %w", err)
	}

	go func() {
		defer w.Close()
		fmt.Fprintf(w, "C0644 %d %s\n", data.Len(), fileName)
		io.Copy(w, data)
		fmt.Fprint(w, "\x00")
	}()

	cmd := "scp -t ."
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("ошибка выполнения SCP: %w", err)
	}
	log.Println("Файл успешно загружен по SCP.")
	return nil
}

// DownloadFile скачивает файл с удаленного сервера, используя SCP
func (c *SSHClient) DownloadFile(fileName string) (*bytes.Buffer, error) {
	client, err := c.connect()
	if err != nil {
		return nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ошибка создания SSH-сессии: %w", err)
	}
	defer session.Close()

	reader, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения stdout: %w", err)
	}

	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		cmd := fmt.Sprintf("scp -f %s", fileName)
		errChan <- session.Run(cmd)
	}()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения данных: %w", err)
	}

	if err := <-errChan; err != nil {
		return nil, fmt.Errorf("ошибка выполнения SCP: %w", err)
	}

	log.Println("Файл успешно скачан по SCP.")
	return &buf, nil
}
