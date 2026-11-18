package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// rmCmd удаляет объект из bucket
var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Удаляет объект из bucket",
	Long: `Команда rm удаляет объект с указанным ключом из bucket.

Пример:
  lazy-s3 rm --url https://s3.example.com --access KEY --secret SECRET \
    --bucket my-bucket --key backup/report.pdf`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRm(rmKey)
	},
}

// флаги команды rm
var rmKey string

func init() {
	rmCmd.Flags().StringVar(&rmKey, "key", "", "ключ объекта в bucket")
	_ = rmCmd.MarkFlagRequired("key")

	addConnectionFlags(rmCmd)
}

// runRm выполняет удаление объекта
func runRm(key string) error {
	conf := newClient()

	if err := conf.Delete(key); err != nil {
		return err
	}

	fmt.Printf("Объект %q удалён из bucket %q\n", key, flagBucket)
	return nil
}
