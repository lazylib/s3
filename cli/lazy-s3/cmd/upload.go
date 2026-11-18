package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

// флаги команды upload
var (
	uploadPath       string
	uploadKey        string
	uploadRecursive  bool
	uploadSkipExists bool
)

// uploadCmd загружает файл или директорию в bucket
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Загружает файл или директорию в bucket",
	Long: `Команда upload загружает данные из локальной файловой системы в bucket.

Если --path указывает на файл — загружается один объект,
--key задаёт точный ключ объекта (по умолчанию — имя файла).

Если --path указывает на директорию — требуется флаг -r (--recursive),
защищающий от случайной загрузки папки. --key задаёт префикс
(по умолчанию — пусто), ключ каждого объекта складывается из префикса
и пути файла относительно директории.

Примеры:
  lazy-s3 upload ... --path ./report.pdf --key backup/report.pdf
  lazy-s3 upload ... --path ./data --key backup -r
  lazy-s3 upload ... --path ./data -r
  lazy-s3 upload ... --path ./data --key backup -r --skip-exists`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpload(uploadPath, uploadKey, uploadRecursive)
	},
}

func init() {
	uploadCmd.Flags().StringVar(&uploadPath, "path", "./", "путь к файлу или директории для загрузки")
	uploadCmd.Flags().BoolVarP(&uploadRecursive, "recursive", "r", false, "рекурсивная загрузка директории (обязателен, если --path — папка)")
	uploadCmd.Flags().BoolVar(&uploadSkipExists, "skip-exists", false, "не загружать файлы, которые уже есть в bucket (проверка по ключу)")
	uploadCmd.Flags().StringVar(&uploadKey, "key", "", "ключ объекта (для файла) или префикс (для директории); по умолчанию — имя файла / пусто")

	addConnectionFlags(uploadCmd)
}

// runUpload определяет тип пути и вызывает загрузку файла или директории
func runUpload(path, key string, recursive bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("не удалось открыть путь %q: %v", path, err)
	}

	if info.IsDir() {
		// защита от случайной загрузки папки: без -r — ошибка с подсказкой
		if !recursive {
			return fmt.Errorf("путь %q — директория: для загрузки папки добавьте флаг -r (--recursive)", path)
		}
		return uploadDir(path, key)
	}

	return uploadFile(path, key, info)
}

// uploadFile загружает один файл; key — точный ключ объекта
func uploadFile(path, key string, info os.FileInfo) error {
	if key == "" {
		// по умолчанию ключ — имя файла
		key = filepath.Base(path)
	} else if strings.HasSuffix(key, "/") {
		// ключ на «папку» — добавляем имя файла
		key = makeKey(key, filepath.Base(path))
	}

	conf := newClient()

	if uploadSkipExists && conf.Exists(key) {
		fmt.Printf("%s -> %s: уже есть в bucket — пропуск\n", path, key)
		return nil
	}

	fmt.Printf("Загрузка %s -> %s (%s) ... ", path, key, humanSize(info.Size()))

	if err := conf.Upload(key, path); err != nil {
		return err
	}

	fmt.Println("OK")
	return nil
}

// uploadDir загружает все файлы из директории; key — префикс
func uploadDir(folder, prefix string) error {
	conf := newClient()

	// собираем список всех файлов в директории
	files, err := listFiles(folder)
	if err != nil {
		return fmt.Errorf("обход директории %q не удался: %v", folder, err)
	}

	if len(files) == 0 {
		fmt.Printf("В директории %q нет файлов для загрузки\n", folder)
		return nil
	}

	fmt.Printf("Загрузка %d файлов из %q в bucket %q (prefix %q)\n\n", len(files), folder, flagBucket, prefix)

	// проход по всем файлам и загрузка через conf
	var (
		loaded  int
		skipped int
		failed  int
		total   int64
	)

	for i, path := range files {
		// ключ объекта — сумма prefix и пути файла относительно folder
		rel, err := filepath.Rel(folder, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%d/%d] %s — ошибка: %v\n", i+1, len(files), path, err)
			failed++
			continue
		}

		key := makeKey(prefix, rel)

		size := int64(0)
		if st, statErr := os.Stat(path); statErr == nil {
			size = st.Size()
		}

		fmt.Printf("[%d/%d] %s -> %s (%s) ... ", i+1, len(files), path, key, humanSize(size))

		if uploadSkipExists && conf.Exists(key) {
			fmt.Println("уже есть в bucket — пропуск")
			skipped++
			continue
		}

		if err := conf.Upload(key, path); err != nil {
			fmt.Printf("ОШИБКА: %v\n", err)
			failed++
			continue
		}

		fmt.Println("OK")
		loaded++
		total += size
	}

	fmt.Printf("\nИтог: загружено %d, пропущено %d, с ошибками %d, объём %s\n", loaded, skipped, failed, humanSize(total))

	if failed > 0 {
		os.Exit(1)
	}

	return nil
}

// listFiles рекурсивно собирает пути всех файлов в директории
func listFiles(folder string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(folder, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files, err
}
