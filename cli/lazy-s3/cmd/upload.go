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

С флагом --skip-exists загружаются только те файлы, которых ещё нет
в bucket: перед загрузкой одним запросом получается список объектов
с указанным префиксом, и файлы с существующими ключами пропускаются.

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
	uploadCmd.Flags().BoolVar(&uploadSkipExists, "skip-exists", false, "не загружать файлы, которые уже есть в bucket (сверка по списку объектов)")
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
		return nil
	}

	fmt.Printf("Загрузка %s -> %s (%s) ... ", path, key, humanSize(info.Size()))

	if err := conf.Upload(key, path); err != nil {
		return err
	}

	fmt.Println("OK")
	return nil
}

// uploadTask — файл, подготовленный к загрузке
type uploadTask struct {
	path string
	key  string
	size int64
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

	// готовим задачи: ключ объекта — сумма prefix и пути файла относительно folder
	tasks := make([]uploadTask, 0, len(files))
	failed := 0
	for _, path := range files {
		rel, err := filepath.Rel(folder, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s — ошибка: %v\n", path, err)
			failed++
			continue
		}

		task := uploadTask{path: path, key: makeKey(prefix, rel)}
		if st, statErr := os.Stat(path); statErr == nil {
			task.size = st.Size()
		}
		tasks = append(tasks, task)
	}

	// с --skip-exists одним запросом получаем список объектов bucket
	// и оставляем только те файлы, которых ещё нет в S3
	skipped := 0
	if uploadSkipExists {
		objects, err := conf.ListAll(strings.Trim(prefix, "/"))
		if err != nil {
			return err
		}

		existing := make(map[string]struct{}, len(objects))
		for _, obj := range objects {
			existing[obj.Key] = struct{}{}
		}

		pending := make([]uploadTask, 0, len(tasks))
		for _, t := range tasks {
			if _, ok := existing[t.key]; ok {
				skipped++
				continue
			}
			pending = append(pending, t)
		}
		tasks = pending
	}

	if len(tasks) > 0 {
		fmt.Printf("Загрузка %d файлов из %q в bucket %q (prefix %q)\n\n", len(tasks), folder, flagBucket, prefix)
	} else if uploadSkipExists {
		fmt.Printf("Все файлы (%d) уже есть в bucket %q (prefix %q) — загружать нечего\n", skipped, flagBucket, strings.Trim(prefix, "/"))
	}

	// загрузка подготовленных задач
	var (
		loaded int
		total  int64
	)

	for i, t := range tasks {
		fmt.Printf("[%d/%d] %s -> %s (%s) ... ", i+1, len(tasks), t.path, t.key, humanSize(t.size))

		if err := conf.Upload(t.key, t.path); err != nil {
			fmt.Printf("ОШИБКА: %v\n", err)
			failed++
			continue
		}

		fmt.Println("OK")
		loaded++
		total += t.size
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
