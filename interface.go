package s3

import "os"

type Config interface {
	GetPresignedDownloadURL(objectKey string) (string, error)
	GetPresignedDownloadURLX(objectKey string) string

	GetPresignedUploadURL(objectKey string, contentType *string) (string, error)
	GetPresignedUploadURLX(objectKey string, contentType *string) string

	Upload(key string, file any) error

	Download(key string, to string) error

	Copy(from, to string) error

	CreateTempFile(objectKey string, pattern string) (*os.File, func() error, error)
}
