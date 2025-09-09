package services

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"package-manager/internal/config"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHClient инкапсулирует логику для работы с SSH-соединением.
// Реализует интерфейс SSHClientInterface
type SSHClient struct {
	config *config.Config
}

// NewSSHClient создает новый экземпляр SSHClient
func NewSSHClient(cfg *config.Config) *SSHClient {
	return &SSHClient{config: cfg}
}

// connect устанавливает SSH-соединение
func (c *SSHClient) connect() (*ssh.Client, error) {
	key, err := os.ReadFile(c.config.SSHKey)
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать SSH-ключ: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("не удалось разобрать SSH-ключ: %w", err)
	}

	knownHostsPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка при чтении файла known_hosts: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User: c.config.SSHUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
	}

	addr := fmt.Sprintf("%s:%d", c.config.SSHHost, c.config.SSHPort)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("ошибка SSH-соединения: %w", err)
	}
	return client, nil
}

// UploadFile загружает файл на удаленный сервер, используя SCP для более простой и быстрой реализации
func (c *SSHClient) UploadFile(fileName string, data *bytes.Buffer) error {
	client, err := c.connect()
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("ошибка создания SSH-сессии: %w", err)
	}
	defer session.Close()

	// Получаем StdinPipe в основной горутине, чтобы избежать гонки
	w, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("ошибка получения StdinPipe: %w", err)
	}

	// Запускаем фоновую горутину для отправки данных
	go func() {
		defer w.Close() // defer тут гарантирует закрытие пайпа после завершения
		fmt.Fprintf(w, "C0644 %d %s\n", data.Len(), fileName)
		io.Copy(w, data)
		fmt.Fprint(w, "\x00")
	}()

	cmd := "scp -t ."
	if err := session.Run(cmd); err != nil {
		return fmt.Errorf("ошибка выполнения SCP: %w", err)
	}
	log.Println("info: Файл успешно загружен по SCP.")
	return nil
}

// DownloadFile скачивает файл с удаленного сервера, используя SCP
func (c *SSHClient) DownloadFile(fileName string) (*bytes.Buffer, error) {
	client, err := c.connect()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("ошибка создания SSH-сессии: %w", err)
	}
	defer session.Close()

	reader, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("ошибка получения stdout: %w", err)
	}

	// Запускаем удаленную команду `scp` в фоновом режиме
	errChan := make(chan error, 1)
	go func() {
		defer close(errChan)
		cmd := fmt.Sprintf("scp -f %s", fileName)
		errChan <- session.Run(cmd)
	}()

	// Читаем данные из пайпа в буфер
	var buf bytes.Buffer
	_, err = io.Copy(&buf, reader)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения данных: %w", err)
	}

	// Дожидаемся завершения удаленной команды
	if err := <-errChan; err != nil {
		return nil, fmt.Errorf("ошибка выполнения SCP: %w", err)
	}

	log.Println("Файл успешно скачан по SCP.")
	return &buf, nil
}
