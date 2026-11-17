package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// флаги команды ls
var (
	lsPrefix string
	lsLimit  int
)

// lsCmd выводит список объектов bucket по префиксу
var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "Выводит список объектов bucket по префиксу",
	Long: `Команда ls выводит список объектов bucket, ключи которых начинаются
с указанного префикса. По умолчанию выводится не более 10 объектов.

Пример:
  lazy-s3 ls --url https://s3.example.com --access KEY --secret SECRET \
    --bucket my-bucket --prefix backup/ --limit 50`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLs(lsPrefix, lsLimit)
	},
}

func init() {
	lsCmd.Flags().StringVar(&lsPrefix, "prefix", "", "prefix для фильтрации объектов")
	lsCmd.Flags().IntVar(&lsLimit, "limit", 10, "максимальное количество объектов в выводе (0 — без лимита)")

	addConnectionFlags(lsCmd)
}

// runLs выполняет запрос списка объектов и выводит их в виде таблицы
func runLs(prefix string, limit int) error {
	conf := newClient()

	objects, err := conf.List(prefix, limit)
	if err != nil {
		return err
	}

	if len(objects) == 0 {
		fmt.Printf("В bucket %q нет объектов с префиксом %q\n", flagBucket, prefix)
		return nil
	}

	// tabwriter выравнивает колонки таблицы
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "ИЗМЕНЁН\tРАЗМЕР\tКЛЮЧ")
	for _, obj := range objects {
		fmt.Fprintf(w, "%s\t%s\t%s\n",
			obj.LastModified.Format(time.RFC3339),
			humanSize(obj.Size),
			obj.Key,
		)
	}
	w.Flush()

	fmt.Printf("\nПоказано: %d\n", len(objects))

	return nil
}
