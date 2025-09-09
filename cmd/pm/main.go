package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"package-manager/internal/config"
	"package-manager/internal/services"
)

var (
	// Корневая команда для CLI-инструмента
	rootCmd = &cobra.Command{
		Use:   "pm",
		Short: "Пакетный менеджер",
		Long:  `Пакетный менеджер для архивации/распаковки и загрузки/скачивания файлов по SSH.`,
	}

	// Команда "pm create"
	createCmd = &cobra.Command{
		Use:   "create [path_to_config]",
		Short: "Упаковывает файлы и загружает на сервер",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadConfig()
			if err != nil {
				log.Fatalf("Error loading configuraton: %v", err)
			}
			sshClient := services.NewSSHClient(cfg)
			pm := services.NewPackageManager(cfg, sshClient)
			if err := pm.CreatePackage(args[0]); err != nil {
				log.Fatalf("Error create to package: %v", err)
			}
		},
	}

	// Команда "pm update"
	updateCmd = &cobra.Command{
		Use:   "update [path_to_config]",
		Short: "Скачивает и распаковывает архивы",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadConfig()
			if err != nil {
				log.Fatalf("Error loading configuraton: %v", err)
			}
			sshClient := services.NewSSHClient(cfg)
			pm := services.NewPackageManager(cfg, sshClient)
			if err := pm.UpdatePackages(args[0]); err != nil {
				log.Fatalf("Error create to package: %v", err)
			}
		},
	}
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Start package manager...")

	rootCmd.AddCommand(createCmd, updateCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
