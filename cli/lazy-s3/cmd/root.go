package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "lazy-s3",
	Short: "Утилита для работы с S3-совместимыми хранилищами",
	Long: `lazy-s3 — утилита командной строки для загрузки и скачивания файлов
в S3-совместимые хранилища через библиотеку github.com/lazylib/s3.

Доступные команды:
  upload   — загружает все файлы из локальной директории в bucket
  download — скачивает объект из bucket в локальный файл`,
	// не выводим usage при ошибках выполнения — он только мешает
	SilenceUsage: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(lsCmd)
}
