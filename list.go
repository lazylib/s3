package s3

import (
	"context"
	"fmt"
	"time"

	s3Lib "github.com/aws/aws-sdk-go-v2/service/s3"
)

// ObjectInfo — информация об объекте в bucket, возвращаемая методом List.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
}

// List возвращает список объектов bucket с указанным префиксом.
// limit ограничивает количество объектов в ответе (0 — без ограничения,
// будет возвращено до 1000 объектов за один запрос).
func (c conf) List(prefix string, limit int) ([]ObjectInfo, error) {
	input := &s3Lib.ListObjectsV2Input{
		Bucket: &c.settings.Bucket,
		Prefix: &prefix,
	}

	if limit > 0 {
		maxKeys := int32(limit)
		input.MaxKeys = &maxKeys
	}

	output, err := c.client.ListObjectsV2(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("не удалось получить список объектов (prefix %q): %v", prefix, err)
	}

	return objectsFromOutput(output), nil
}

// ListAll возвращает полный список объектов bucket с указанным префиксом.
// В отличие от List, метод автоматически проходит все страницы ответа
// (по 1000 объектов на запрос), поэтому возвращает все подходящие объекты.
func (c conf) ListAll(prefix string) ([]ObjectInfo, error) {
	input := &s3Lib.ListObjectsV2Input{
		Bucket: &c.settings.Bucket,
		Prefix: &prefix,
	}

	paginator := s3Lib.NewListObjectsV2Paginator(c.client, input)

	objects := make([]ObjectInfo, 0)
	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return nil, fmt.Errorf("не удалось получить список объектов (prefix %q): %v", prefix, err)
		}
		objects = append(objects, objectsFromOutput(output)...)
	}

	return objects, nil
}

// objectsFromOutput преобразует ответ ListObjectsV2 в срез ObjectInfo.
func objectsFromOutput(output *s3Lib.ListObjectsV2Output) []ObjectInfo {
	objects := make([]ObjectInfo, 0, len(output.Contents))
	for _, obj := range output.Contents {
		info := ObjectInfo{}
		if obj.Key != nil {
			info.Key = *obj.Key
		}
		if obj.Size != nil {
			info.Size = *obj.Size
		}
		if obj.LastModified != nil {
			info.LastModified = *obj.LastModified
		}
		objects = append(objects, info)
	}
	return objects
}
