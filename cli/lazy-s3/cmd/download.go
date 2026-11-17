package cmd

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lazylib/s3"
	"github.com/spf13/cobra"
)

// флаги команды download
var (
	downloadKey     string
	downloadPath    string
	downloadWorkers int
)

// downloadCmd скачивает объект или группу объектов из bucket
var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "Скачивает объект или группу объектов из bucket",
	Long: `Команда download скачивает данные из bucket в локальную файловую систему.

Если передан --key:
  - ключ на объект  — скачивается один файл;
  - ключ на «папку» — используется как префикс, скачиваются все объекты,
    у которых ключ начинается с этого префикса.

Если --key не передан — скачивается весь bucket.

Примеры:
  lazy-s3 download ... --key backup/report.pdf --path ./report.pdf
  lazy-s3 download ... --key backup/ --path ./backup
  lazy-s3 download ... --path ./dump`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDownload(downloadKey, downloadPath, downloadWorkers)
	},
}

func init() {
	downloadCmd.Flags().StringVar(&downloadKey, "key", "", "ключ объекта или префикс (папка) в bucket; пусто — весь bucket")
	downloadCmd.Flags().StringVar(&downloadPath, "path", "./", "локальный путь для сохранения")
	downloadCmd.Flags().IntVar(&downloadWorkers, "workers", 1, "количество параллельных загрузок при скачивании нескольких объектов")

	addConnectionFlags(downloadCmd)
}

// runDownload выполняет скачивание объекта, префикса или всего bucket
func runDownload(key, localPath string, workers int) error {
	if key != "" && !strings.HasSuffix(key, "/") {
		// возможно это одиночный объект — проверяем листингом с лимитом 1
		conf := newClient()
		objects, err := conf.List(key, 1)
		if err != nil {
			return err
		}

		if len(objects) == 1 && objects[0].Key == key {
			// это действительно один объект
			if strings.HasSuffix(localPath, "/") || localPath == "" {
				localPath = path.Join(localPath, path.Base(key))
			}
			if err := conf.Download(key, localPath); err != nil {
				return err
			}
			fmt.Printf("Объект %q из bucket %q сохранён в %q\n", key, flagBucket, localPath)
			return nil
		}
	}

	// иначе — скачивание группы объектов: key как префикс (или весь bucket)
	prefix := strings.Trim(key, "/")
	if prefix != "" {
		prefix += "/"
	}

	return downloadMany(prefix, localPath, workers)
}

type downloadResult struct {
	index int
	key   string
	dst   string
	size  int64
	err   error
}

// downloadMany скачивает все объекты с префиксом prefix в localPath
func downloadMany(prefix, localPath string, workers int) error {
	conf := newClient()

	objects, err := conf.ListAll(prefix)
	if err != nil {
		return err
	}

	if len(objects) == 0 {
		if prefix != "" {
			fmt.Printf("В bucket %q нет объектов с префиксом %q\n", flagBucket, prefix)
		} else {
			fmt.Printf("Bucket %q пуст\n", flagBucket)
		}
		return nil
	}

	// отфильтровываем «маркеры папок» — объекты с ключами, заканчивающимися
	// на "/" (нулевого размера): локально это директории, скачивать их не нужно
	files := make([]s3.ObjectInfo, 0, len(objects))
	skippedDirs := 0
	for _, obj := range objects {
		if strings.HasSuffix(obj.Key, "/") {
			skippedDirs++
			continue
		}
		files = append(files, obj)
	}

	target := localPath
	if strings.HasSuffix(localPath, "/") || localPath == "" {
		// сохраняем относительно пути: backup/a.txt -> ./backup/a.txt
		target = path.Join(localPath, prefix)
	}
	target = strings.TrimSuffix(target, "/")

	fmt.Printf("Скачивание %d объектов из bucket %q (prefix %q) в %q\n\n",
		len(files), flagBucket, prefix, target)

	results := make(chan downloadResult, len(files))

	var (
		mu         sync.Mutex
		wg         sync.WaitGroup
		sem        = make(chan struct{}, workers)
		downloaded int
		failed     int
		total      int64
		failures   []string
	)

	for i, obj := range files {
		wg.Add(1)
		go func(i int, obj s3.ObjectInfo) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// локальный путь = target + часть ключа после префикса
			rel := strings.TrimPrefix(obj.Key, prefix)
			dst := filepath.Join(target, filepath.FromSlash(rel))

			res := downloadResult{index: i, key: obj.Key, dst: dst, size: obj.Size}

			if err := conf.Download(obj.Key, dst); err != nil {
				res.err = err
				mu.Lock()
				failed++
				failures = append(failures, fmt.Sprintf("%s -> %s: %v", obj.Key, dst, err))
				mu.Unlock()
				results <- res
				return
			}

			mu.Lock()
			downloaded++
			total += obj.Size
			mu.Unlock()
			results <- res
		}(i, obj)
	}

	// закрываем канал после завершения всех воркеров
	go func() {
		wg.Wait()
		close(results)
	}()

	// печать по мере поступления — без разрывов строк, атомарно
	for res := range results {
		if res.err != nil {
			fmt.Printf("[%d/%d] %s — ОШИБКА: %v\n", res.index+1, len(files), res.key, res.err)
			continue
		}
		fmt.Printf("[%d/%d] %s -> %s (%s)\n", res.index+1, len(files), res.key, res.dst, humanSize(res.size))
	}

	if skippedDirs > 0 {
		fmt.Printf("\nПропущено папок-маркеров: %d\n", skippedDirs)
	}

	fmt.Printf("Итог: скачано %d, с ошибками %d, объём %s\n", downloaded, failed, humanSize(total))

	if len(failures) > 0 {
		fmt.Fprintf(os.Stderr, "\nНе скачались (%d):\n", len(failures))
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "  %s\n", f)
		}
		os.Exit(1)
	}

	return nil
}
