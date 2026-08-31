package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3Lib "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Upload загружает файл в S3 под ключом key напрямую (без presigned-ссылок).
// Параметр file может быть:
//   - string    — путь к файлу на диске
//   - []byte    — содержимое файла
//   - io.Reader — поток данных (в т.ч. *os.File)
func (c conf) Upload(key string, file any) error {
	reader, size, err := toReader(file)
	if err != nil {
		return fmt.Errorf("не удалось подготовить файл для загрузки: %v", err)
	}
	defer func() {
		if closer, ok := reader.(io.Closer); ok {
			_ = closer.Close()
		}
	}()

	contentType := mime.TypeByExtension(filepath.Ext(key))

	input := &s3Lib.PutObjectInput{
		Bucket:      &c.settings.Bucket,
		Key:         &key,
		Body:        reader,
		ContentType: aws.String(contentType),
	}

	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	if _, err := c.client.PutObject(context.TODO(), input); err != nil {
		return fmt.Errorf("не удалось загрузить объект %q: %v", key, err)
	}

	return nil
}

// toReader приводит произвольный файл к io.Reader и возвращает его размер
// (0 — если размер заранее неизвестен).
func toReader(file any) (io.Reader, int64, error) {
	switch f := file.(type) {
	case string:
		opened, err := os.Open(f)
		if err != nil {
			return nil, 0, err
		}
		info, err := opened.Stat()
		if err != nil {
			_ = opened.Close()
			return nil, 0, err
		}
		return opened, info.Size(), nil
	case []byte:
		return bytes.NewReader(f), int64(len(f)), nil
	case io.Reader:
		return f, 0, nil
	default:
		return nil, 0, fmt.Errorf("неподдерживаемый тип файла: %T", file)
	}
}
