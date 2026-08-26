package s3

import (
	"context"
	"fmt"
	"io"
	"os"

	s3Lib "github.com/aws/aws-sdk-go-v2/service/s3"
)

// CreateTempFile скачивает объект из S3 во временный файл и возвращает
// ссылку на него вместе с функцией remove, которую удобно вызывать через defer:
//
//	f, remove, err := c.CreateTempFile(objectKey)
//	if err != nil {
//	    return err
//	}
//	defer remove()
//
// Важно: если файл не был удалён через remove(), он останется в системной
// временной директории (и будет удалён ОС позже, но лучше чистить самим).
func (c conf) CreateTempFile(objectKey string) (*os.File, func() error, error) {
	// сначала создаём временный файл, чтобы при ошибке скачивания не оставлять мусор
	tmp, err := os.CreateTemp("", "s3-*")
	if err != nil {
		return nil, nil, fmt.Errorf("не удалось создать временный файл: %v", err)
	}

	// удаляем файл, если что-то пошло не так до возврата из функции
	cleanupOnError := func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}

	// скачиваем объект через обычный клиент (не presigned)
	input := &s3Lib.GetObjectInput{
		Bucket: &c.settings.Bucket,
		Key:    &objectKey,
	}

	obj, err := c.client.GetObject(context.TODO(), input)
	if err != nil {
		cleanupOnError()
		return nil, nil, fmt.Errorf("не удалось скачать объект %q: %v", objectKey, err)
	}
	defer obj.Body.Close()

	// копируем тело объекта в файл
	if _, err := io.Copy(tmp, obj.Body); err != nil {
		cleanupOnError()
		return nil, nil, fmt.Errorf("не удалось записать объект %q во временный файл: %v", objectKey, err)
	}

	// перематываем на начало, чтобы вызывающий код мог сразу читать
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		cleanupOnError()
		return nil, nil, fmt.Errorf("не удалось перемотать временный файл: %v", err)
	}

	// функция удаления для defer у вызывающего кода
	remove := func() error {
		// закрываем файл (если ещё не закрыт) и удаляем с диска
		_ = tmp.Close()
		if err := os.Remove(tmp.Name()); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("не удалось удалить временный файл %q: %v", tmp.Name(), err)
		}
		return nil
	}

	return tmp, remove, nil
}
