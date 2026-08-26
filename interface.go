package s3

import "os"

type Config interface {
	GetPresignedDownloadURL(objectKey string) (string, error)
	GetPresignedDownloadURLX(objectKey string) string

	GetPresignedUploadURL(objectKey string, contentType *string) (string, error)
	GetPresignedUploadURLX(objectKey string, contentType *string) string

	CreateTempFile(objectKey string) (*os.File, func() error, error)
}
