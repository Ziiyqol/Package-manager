package services

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"package-manager/internal/config"
)

// MockSSHClient мок для тестирования, реализует интерфейс SSHClientInterface
type MockSSHClient struct {
	UploadFileFunc   func(fileName string, data *bytes.Buffer) error
	DownloadFileFunc func(fileName string) (*bytes.Buffer, error)
}

func (m *MockSSHClient) UploadFile(fileName string, data *bytes.Buffer) error {
	return m.UploadFileFunc(fileName, data)
}

func (m *MockSSHClient) DownloadFile(fileName string) (*bytes.Buffer, error) {
	return m.DownloadFileFunc(fileName)
}

// TestCreatePackageWithMockClient тестирует создание пакета, используя мок-объект SSH-клиента
func TestCreatePackageWithMockClient(t *testing.T) {
	// Временная директория и файлы для тестирования
	tempDir, err := os.MkdirTemp("", "test-pm")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello, GoGoGooolang!"), 0644); err != nil {
		t.Fatalf("Не удалось создать тестовый файл: %v", err)
	}

	// Создаем тестовый packet.json
	configFile := filepath.Join(tempDir, "packet.json")
	configData := []byte(`{
		"name": "test-pkg",
		"ver": "1.0", 
		"targets": [{"path": "./test.txt"}]
	}`)
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		t.Fatalf("Не удалось создать файл конфигурации: %v", err)
	}

	// Сохраняем рабочую директорию, чтобы вернуться в нее после теста
	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Не удалось получить текущую рабочую директорию: %v", err)
	}
	// Изменяем рабочую директорию на временную для теста
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Не удалось изменить рабочую директорию: %v", err)
	}
	// Убеждаемся, что мы вернемся в исходную директорию, когда тест завершится
	defer os.Chdir(currentDir)

	// Создаем мок-объект SSH-клиента
	mockSSHClient := &MockSSHClient{
		UploadFileFunc: func(fileName string, data *bytes.Buffer) error {
			// Проверяем, что файл был передан с правильным именем
			if fileName != "test-pkg-1.0.zip" {
				t.Errorf("Ожидалось имя файла 'test-pkg-1.0.zip', получено '%s'", fileName)
				return errors.New("неверное имя файла")
			}
			// Проверяем, что архив не пустой
			if data.Len() == 0 {
				t.Error("Ожидался заполненный архив, но он пуст")
				return errors.New("пустой архив")
			}
			return nil
		},
	}

	// Создаем PackageManager, используя мок SSH-клиента
	pm := NewPackageManager(&config.Config{}, mockSSHClient)

	// Проверяем, что вызов `CreatePackage` не приводит к ошибке
	err = pm.CreatePackage(filepath.Base(configFile))
	if err != nil {
		t.Errorf("Ожидалась успешная упаковка, но получена ошибка: %v", err)
	}
}

// TestUpdatePackagesWithMockClient тестирует обновление пакетов
func TestUpdatePackagesWithMockClient(t *testing.T) {
	// Создаем временную директорию и файл конфигурации
	tempDir, err := os.MkdirTemp("", "test-update")
	if err != nil {
		t.Fatalf("Не удалось создать временную директорию: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Создаем тестовый packages.json
	configFile := filepath.Join(tempDir, "packages.json")
	configData := []byte(`{
		"packages": [
		{"name": "test-pkg", 
		"ver": "1.0"}
		]
	}`)
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		t.Fatalf("Не удалось создать файл конфигурации: %v", err)
	}

	// Создаем мок SSH-клиента
	mockSSHClient := &MockSSHClient{
		DownloadFileFunc: func(fileName string) (*bytes.Buffer, error) {
			// Проверяем, что запрашивается правильное имя файла
			if fileName != "test-pkg-1.0.zip" {
				return nil, errors.New("неверное имя файла")
			}
			// Возвращаем пустой буфер для симуляции скачивания
			return bytes.NewBuffer([]byte{}), nil
		},
	}

	// Создаем PackageManager, используя мок SSH-Клиента
	pm := NewPackageManager(&config.Config{}, mockSSHClient)

	// Проверяем, что вызов `UpdatePackages` не приводит к ошибке
	err = pm.UpdatePackages(configFile)
	if err != nil {
		t.Errorf("Ожидалось успешное обновление, но получена ошибка: %v", err)
	}
}
