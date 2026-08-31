package s3

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	s3Lib "github.com/aws/aws-sdk-go-v2/service/s3"
)

// Copy копирует объект из from в to внутри одного бакета.
// Копирование выполняется на стороне S3 (CopyObject), данные не скачиваются.
func (c conf) Copy(from, to string) error {
	input := &s3Lib.CopyObjectInput{
		Bucket:     &c.settings.Bucket,
		CopySource: aws.String(fmt.Sprintf("%s/%s", c.settings.Bucket, from)),
		Key:        &to,
	}

	_, err := c.client.CopyObject(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("не удалось скопировать объект %q в %q: %v", from, to, err)
	}

	return nil
}
