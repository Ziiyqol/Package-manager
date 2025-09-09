package services

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"package-manager/internal/config"
	"package-manager/internal/models"
)

// PackageManager (далее PM) содержит логику для создания и обновления пакетов
type PackageManager struct {
	config    *config.Config
	sshClient SSHClientInterface
}

// NewPackageManager создает новый экземпляр PM с SSH-клиентом
func NewPackageManager(cfg *config.Config, sshClient SSHClientInterface) *PackageManager {
	return &PackageManager{config: cfg, sshClient: sshClient}
}

// ReadConfig читает и парсит файл конфигурации
func (pm *PackageManager) ReadConfig(path string, cfg any) error {
	ext := filepath.Ext(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("ошибка чтения файла: %w", err)
	}

	switch strings.ToLower(ext) {
	case ".json":
		return json.Unmarshal(data, cfg)
	case ".yaml", ".yml":
		return yaml.Unmarshal(data, cfg)
	default:
		return fmt.Errorf("неподдерживаемый формат файла: %s", ext)
	}
}

// CreatePackage упаковывает файлы и загружает их на сервер
func (pm *PackageManager) CreatePackage(configPath string) error {
	var cfg models.CreateConfig
	if err := pm.ReadConfig(configPath, &cfg); err != nil {
		return fmt.Errorf("ошибка парсинга файла %s: %w", configPath, err)
	}

	log.Printf("Создание пакета %s версии %s...", cfg.Name, cfg.Ver)

	// Создаем временный ZIP-архив в памяти
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	for _, target := range cfg.Targets {
		matches, err := filepath.Glob(target.Path)
		if err != nil {
			log.Printf("Ошибка при поиске файлов по маске %s: %v", target.Path, err)
			continue
		}

		for _, match := range matches {
			info, err := os.Stat(match)
			if err != nil {
				log.Printf("Не удалось получить информацию о файле %s: %v", match, err)
				continue
			}

			if info.IsDir() {
				// Рекурсивное добавление содержимого директории
				if err := pm.addDirToArchive(zipWriter, match, target.Exclude); err != nil {
					log.Printf("Не удалось добавить директорию %s в архив: %v", match, err)
				}
			} else {
				// Добавление одиночного файла
				if err := pm.addFileToArchive(zipWriter, match, target.Exclude); err != nil {
					log.Printf("Не удалось добавить файл %s в архив: %v", match, err)
				}
			}
		}
	}

	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("ошибка закрытия архива: %w", err)
	}
	log.Printf("Архив создан, размер: %d байт.", buf.Len())

	// Загружаем архив на сервер по SSH, используя внедренный клиент
	archiveName := fmt.Sprintf("%s-%s.zip", cfg.Name, cfg.Ver)
	if err := pm.sshClient.UploadFile(archiveName, buf); err != nil {
		return fmt.Errorf("ошибка загрузки пакета по SSH: %w", err)
	}

	log.Printf("Пакет %s успешно загружен на сервер.", archiveName)
	return nil
}

// UpdatePackages скачивает и распаковывает архивы с сервера
func (pm *PackageManager) UpdatePackages(configPath string) error {
	var cfg models.UpdateConfig
	if err := pm.ReadConfig(configPath, &cfg); err != nil {
		return fmt.Errorf("ошибка парсинга файла %s: %w", configPath, err)
	}

	log.Println("Обновление пакетов...")

	for _, pkg := range cfg.Packages {
		archiveName := fmt.Sprintf("%s-%s.zip", pkg.Name, pkg.Ver)
		log.Printf("Скачивание и распаковка пакета %s...", archiveName)

		buf, err := pm.sshClient.DownloadFile(archiveName)
		if err != nil {
			log.Printf("Не удалось скачать пакет %s: %v", archiveName, err)
			continue
		}

		// Распаковка архива
		zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
		if err != nil {
			log.Printf("Ошибка создания ZIP-ридера для %s: %v", archiveName, err)
			continue
		}

		for _, f := range zipReader.File {
			path := filepath.Join(".", f.Name)
			if f.FileInfo().IsDir() {
				os.MkdirAll(path, f.Mode())
				continue
			}

			os.MkdirAll(filepath.Dir(path), 0755)
			outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				log.Printf("Ошибка создания файла %s: %v", path, err)
				continue
			}
			rc, err := f.Open()
			if err != nil {
				outFile.Close()
				log.Printf("Ошибка открытия файла в архиве %s: %v", f.Name, err)
				continue
			}
			_, err = io.Copy(outFile, rc)
			outFile.Close()
			rc.Close()

			if err != nil {
				log.Printf("Ошибка распаковки файла %s: %v", f.Name, err)
			}
			log.Printf("Распакован файл: %s", path)
		}
		log.Printf("Пакет %s успешно распакован.", pkg.Name)
	}

	return nil
}

// addFileToArchive добавляет одиночный файл в архив, если он не исключен
func (pm *PackageManager) addFileToArchive(writer *zip.Writer, filePath string, exclude string) error {
	// Проверка на исключение
	if exclude != "" {
		excludeMatch, err := filepath.Match(exclude, filepath.Base(filePath))
		if err != nil {
			return fmt.Errorf("ошибка при проверке исключения %s для файла %s: %w", exclude, filePath, err)
		}
		if excludeMatch {
			log.Printf("Исключение файла %s", filePath)
			return nil
		}
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл %s: %w", filePath, err)
	}
	defer file.Close()

	zipFile, err := writer.Create(filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("не удалось создать запись в архиве для файла %s: %w", filePath, err)
	}

	_, err = io.Copy(zipFile, file)
	if err != nil {
		return fmt.Errorf("не удалось скопировать данные в архив из файла %s: %w", filePath, err)
	}
	return nil
}

// addDirToArchive рекурсивно добавляет директорию в архив
func (pm *PackageManager) addDirToArchive(writer *zip.Writer, dirPath string, exclude string) error {
	// Исключаем саму директорию, если она совпадает с исключениями
	if exclude != "" {
		excludeMatch, err := filepath.Match(exclude, filepath.Base(dirPath))
		if err != nil {
			return fmt.Errorf("ошибка при проверке исключения %s для папки %s: %w", exclude, dirPath, err)
		}
		if excludeMatch {
			log.Printf("Исключение директории %s", dirPath)
			return filepath.SkipDir
		}
	}

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Создаем заголовок файла в архиве
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Устанавливаем относительный путь
		relPath, err := filepath.Rel(filepath.Dir(dirPath), path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if info.IsDir() {
			header.Name += "/"
			// Пропускаем исключенные поддиректории
			if exclude != "" {
				excludeMatch, err := filepath.Match(exclude, filepath.Base(path))
				if err != nil {
					return fmt.Errorf("ошибка при проверке исключения %s для папки %s: %w", exclude, path, err)
				}
				if excludeMatch {
					log.Printf("Исключение поддиректории %s", path)
					return filepath.SkipDir
				}
			}
		} else {
			// Пропускаем исключенные файлы
			if exclude != "" {
				excludeMatch, err := filepath.Match(exclude, filepath.Base(path))
				if err != nil {
					return fmt.Errorf("ошибка при проверке исключения %s для файла %s: %w", exclude, path, err)
				}
				if excludeMatch {
					log.Printf("Исключение файла %s", path)
					return nil
				}
			}
			header.Method = zip.Deflate
		}

		writer, err := writer.CreateHeader(header)
		if err != nil {
			return fmt.Errorf("не удалось создать запись в архиве для %s: %w", path, err)
		}

		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("не удалось открыть файл %s: %w", path, err)
			}
			defer file.Close()
			_, err = io.Copy(writer, file)
			if err != nil {
				return fmt.Errorf("не удалось скопировать данные в архив из файла %s: %w", path, err)
			}
		}
		return nil
	})
}
