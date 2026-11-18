# s3

A small, convenient Go wrapper around the AWS S3 SDK v2 (`aws-sdk-go-v2`) for common S3 operations: presigned URLs, upload, download, server-side copy, and temp files.

## Features

- Minimal configuration — one struct, one constructor
- Presigned **upload** and **download** URLs (with time-to-live limits)
- Direct `Upload` that accepts a file path, `[]byte`, or `io.Reader`
- `Download` to a local path (creates parent directories automatically)
- Server-side `Copy` inside a bucket (no data transfer through your app)
- `CreateTempFile` for safe "download to temp file" flows with a cleanup function
- Panic-free variants (`...X` suffix) that return the URL directly or panic

## Installation

```bash
go get github.com/lazylib/s3
```

## Usage

### Creating the config

The library is initialized with the `New(settings Settings)` constructor. Call it once at startup and reuse the returned `Config` everywhere.

```go
package main

import (
	"log"
	"time"

	"github.com/lazylib/s3"
)

func init() {
	// ...or assign to a package-level variable:
	// s3Client = s3.New(...)
}

var s3Client s3.Config

func main() {
	s3Client = s3.New(s3.Settings{
		Url:       "https://s3.example.com",
		Region:    "ru-1", // default if empty
		Bucket:    "my-bucket",
		AccessKey: "access-key",
		SecretKey: "secret-key",

		// Path-style addressing (e.g. http://host/bucket/key).
		// Set to true for most self-hosted / MinIO endpoints.
		UsePathStyle: true,

		Upload: &s3.LinkSettings{
			LiveTimeDuration: 15 * time.Minute, // presigned URL lifetime
			MaxSizeByte:      200 * 1024,       // max upload size
		},
		Download: &s3.LinkSettings{
			LiveTimeDuration: 15 * time.Minute,
			MaxSizeByte:      200 * 1024,
		},
	})

	log.Println(s3Client) // use it
}
```

> If `Upload` or `Download` is `nil`, the default `LinkSettings` are used:
> `LiveTimeDuration: 15 * time.Minute`, `MaxSizeByte: 200 * 1024`.

### Presigned URLs

Upload URL — content type is auto-detected from the file extension when `nil` is passed:

```go
url, err := s3Client.GetPresignedUploadURL("images/photo.jpg", nil)
if err != nil {
	log.Fatal(err)
}

// Same, but panics on error:
url = s3Client.GetPresignedUploadURLX("images/photo.jpg", nil)
```

Download URL — forces `attachment` content disposition (download, not inline):

```go
url, err := s3Client.GetPresignedDownloadURL("images/photo.jpg")
if err != nil {
	log.Fatal(err)
}

url = s3Client.GetPresignedDownloadURLX("images/photo.jpg")
```

### Upload

`file` can be a `string` (path on disk), `[]byte`, or any `io.Reader` (including `*os.File`):

```go
// By path
err := s3Client.Upload("reports/2024.pdf", "/tmp/2024.pdf")

// From bytes
err = s3Client.Upload("data/blob.bin", []byte{0xde, 0xad, 0xbe, 0xef})

// From a stream
f, _ := os.Open("photo.jpg")
defer f.Close()
err = s3Client.Upload("images/photo.jpg", f)
```

### Download

```go
err := s3Client.Download("reports/2024.pdf", "/tmp/2024.pdf")
// Parent directories are created automatically.
```

### Copy (server-side, within one bucket)

```go
err := s3Client.Copy("reports/2024.pdf", "archive/2024.pdf")
```

### Rename (server-side, within one bucket)

There is no rename operation in S3, so the object is copied on the S3 side and
then the original is deleted:

```go
err := s3Client.Rename("reports/2024.pdf", "reports/2025.pdf")
// Renaming to the same key is a no-op.
```

### Temp file

Downloads an object into a system temp file and returns it with a `remove` function — perfect for `defer`:

```go
f, remove, err := s3Client.CreateTempFile("reports/2024.pdf", "report-*.pdf")
if err != nil {
	log.Fatal(err)
}
defer remove() // removes the temp file after processing

data, _ := io.ReadAll(f)
```

## Config interface

```go
type Config interface {
	GetPresignedDownloadURL(objectKey string) (string, error)
	GetPresignedDownloadURLX(objectKey string) string

	GetPresignedUploadURL(objectKey string, contentType *string) (string, error)
	GetPresignedUploadURLX(objectKey string, contentType *string) string

	Upload(key string, file any) error
	Download(key string, to string) error
	Copy(from, to string) error

	Rename(oldname, newname string) error

	CreateTempFile(objectKey string, pattern string) (*os.File, func() error, error)
}
```

## License

[MIT](LICENSE)
