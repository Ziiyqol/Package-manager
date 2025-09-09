package services

import (
	"bytes"
)

// SSHClientInterface определяет контракт для работы с SSH-клиентом для тестов
type SSHClientInterface interface {
	UploadFile(fileName string, data *bytes.Buffer) error
	DownloadFile(fileName string) (*bytes.Buffer, error)
}
