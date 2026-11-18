package s3

import (
	"context"

	s3Lib "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Exists проверяет наличие объекта key в bucket (HeadObject).
// Если объекта нет — возвращает false. Обратите внимание: ошибки запроса
// (сеть, доступ и т.п.) также приводят к false, т.к. метод не возвращает
// ошибку.
func (c conf) Exists(key string) bool {
	input := &s3Lib.HeadObjectInput{
		Bucket: &c.settings.Bucket,
		Key:    &key,
	}

	_, err := c.client.HeadObject(context.TODO(), input)
	return err == nil
}
