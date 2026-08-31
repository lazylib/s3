package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	s3Lib "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Download скачивает объект из S3 по ключу key и сохраняет его
// в указанное место to (например "/tmp/file.pdf").
func (c conf) Download(key string, to string) error {
	// создаём директорию назначения, если её нет
	if dir := filepath.Dir(to); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("не удалось создать директорию %q: %v", dir, err)
		}
	}

	// создаём файл назначения
	file, err := os.Create(to)
	if err != nil {
		return fmt.Errorf("не удалось создать файл %q: %v", to, err)
	}
	defer file.Close()

	// скачиваем объект через обычный клиент (не presigned)
	input := &s3Lib.GetObjectInput{
		Bucket: &c.settings.Bucket,
		Key:    &key,
	}

	obj, err := c.client.GetObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("не удалось скачать объект %q: %v", key, err)
	}
	defer obj.Body.Close()

	// копируем тело объекта в файл
	if _, err := io.Copy(file, obj.Body); err != nil {
		return fmt.Errorf("не удалось записать объект %q в файл %q: %v", key, to, err)
	}

	return nil
}
