package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/lazylib/s3"
)

func main() {
	folder := flag.String("folder", "./", "путь к директории которую нужно загрузить")
	url := flag.String("url", "", "Url s3")
	access := flag.String("access", "", "accesskey")
	secret := flag.String("secret", "", "secretkey")
	bucket := flag.String("bucket", "", "bucket")
	pathstyle := flag.Bool("pathstyle", true, "использовать pathstyle по умолчанию true")
	prefix := flag.String("prefix", "", "prefix для целевого пути")

	flag.Parse()

	// проверка аргументов: выводим понятную ошибку и готовую команду
	// с предзаполненными параметрами, чтобы её можно было скопировать и использовать
	missing := make([]string, 0, 4)
	if *url == "" {
		missing = append(missing, "-url")
	}
	if *access == "" {
		missing = append(missing, "-access")
	}
	if *secret == "" {
		missing = append(missing, "-secret")
	}
	if *bucket == "" {
		missing = append(missing, "-bucket")
	}

	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "Ошибка: не заданы обязательные параметры: %s\n\n", strings.Join(missing, ", "))
		fmt.Fprintf(os.Stderr, "Готовая команда (заполните пустые значения в кавычках):\n\n")
		fmt.Fprintf(os.Stderr, "  s3util -url %q -access %q -secret %q -bucket %q -folder %q -prefix %q -pathstyle=%t\n\n",
			*url, *access, *secret, *bucket, *folder, *prefix, *pathstyle)
		os.Exit(1)
	}

	info, err := os.Stat(*folder)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: не удалось открыть папку %q: %v\n", *folder, err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Ошибка: путь %q не является директорией\n", *folder)
		os.Exit(1)
	}

	conf := s3.New(s3.Settings{
		Url:          *url,
		AccessKey:    *access,
		Region:       "ru",
		Bucket:       *bucket,
		SecretKey:    *secret,
		UsePathStyle: *pathstyle,
	})

	// собираем список всех файлов в директории
	var files []string
	err = filepath.WalkDir(*folder, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: обход директории %q не удался: %v\n", *folder, err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Printf("В директории %q нет файлов для загрузки\n", *folder)
		return
	}

	fmt.Printf("Загрузка %d файлов из %q в bucket %q\n\n", len(files), *folder, *bucket)

	// проход по всем файлам и загрузка через conf
	var (
		loaded int
		failed int
		total  int64
	)

	for i, path := range files {
		// key — сумма prefix и пути до файла относительно folder
		rel, err := filepath.Rel(*folder, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%d/%d] %s — ошибка: %v\n", i+1, len(files), path, err)
			failed++
			continue
		}

		key := makeKey(*prefix, rel)

		size := int64(0)
		if st, statErr := os.Stat(path); statErr == nil {
			size = st.Size()
		}

		fmt.Printf("[%d/%d] %s -> %s (%s) ... ", i+1, len(files), path, key, humanSize(size))

		if err := conf.Upload(key, path); err != nil {
			fmt.Printf("ОШИБКА: %v\n", err)
			failed++
			continue
		}

		fmt.Println("OK")
		loaded++
		total += size
	}

	fmt.Printf("\nИтог: загружено %d, с ошибками %d, объём %s\n", loaded, failed, humanSize(total))

	if failed > 0 {
		os.Exit(1)
	}
}

// makeKey соединяет prefix и относительный путь в ключ объекта S3,
// нормализуя разделители к "/".
func makeKey(prefix, rel string) string {
	rel = filepath.ToSlash(rel)
	prefix = strings.Trim(filepath.ToSlash(prefix), "/")
	if prefix == "" {
		return rel
	}
	return prefix + "/" + rel
}

// humanSize форматирует размер в человекочитаемом виде.
func humanSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	value := float64(size)
	units := []string{"KB", "MB", "GB", "TB"}
	for _, u := range units {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, u)
		}
	}

	return fmt.Sprintf("%.1f PB", value/unit)
}
