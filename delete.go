package s3

import (
	"context"
	"fmt"

	s3Lib "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Delete удаляет объект key из bucket.
// Операция выполняется на стороне S3 (DeleteObject).
func (c conf) Delete(key string) error {
	input := &s3Lib.DeleteObjectInput{
		Bucket: &c.settings.Bucket,
		Key:    &key,
	}

	if _, err := c.client.DeleteObject(context.TODO(), input); err != nil {
		return fmt.Errorf("не удалось удалить объект %q: %v", key, err)
	}

	return nil
}
