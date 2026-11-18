package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/lazylib/s3"
	"github.com/spf13/cobra"
)

// общие флаги подключения к S3, используются командами upload и download
var (
	flagURL       string
	flagAccess    string
	flagSecret    string
	flagBucket    string
	flagRegion    string
	flagPathStyle bool
)

// addConnectionFlags добавляет к команде общие флаги подключения к S3
func addConnectionFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flagURL, "url", "", "url s3-хранилища")
	cmd.Flags().StringVar(&flagAccess, "access", "", "access key")
	cmd.Flags().StringVar(&flagSecret, "secret", "", "secret key")
	cmd.Flags().StringVar(&flagBucket, "bucket", "", "имя bucket")
	cmd.Flags().StringVar(&flagRegion, "region", "ru", "регион s3")
	cmd.Flags().BoolVar(&flagPathStyle, "pathstyle", true, "использовать pathstyle (по умолчанию true)")

	// параметры подключения обязательны для любой команды
	for _, name := range []string{"url", "access", "secret", "bucket"} {
		_ = cmd.MarkFlagRequired(name)
	}
}

// newClient создаёт клиент s3 на основе общих флагов подключения
func newClient() s3.Config {
	return s3.New(s3.Settings{
		Url:          flagURL,
		AccessKey:    flagAccess,
		Region:       flagRegion,
		Bucket:       flagBucket,
		SecretKey:    flagSecret,
		UsePathStyle: flagPathStyle,
		// хранилища без поддержки контрольных сумм порождают
		// поток WARN-сообщений SDK, глушим их
		DisableChecksumWarningLog: true,
	})
}

// makeKey соединяет prefix и относительный путь в ключ объекта S3,
// нормализуя разделители к "/".
func makeKey(prefix, rel string) string {
	rel = toSlash(rel)
	prefix = trimSlash(toSlash(prefix))
	if prefix == "" {
		return rel
	}
	return prefix + "/" + rel
}

// humanSize форматирует размер в человекочитаемом виде.
func humanSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}

	value := float64(size)
	units := []string{"KB", "MB", "GB", "TB"}
	for _, u := range units {
		value /= unit
		if value < unit {
			return fmt.Sprintf("%.1f %s", value, u)
		}
	}

	return fmt.Sprintf("%.1f PB", value/unit)
}

// toSlash заменяет разделители пути на "/"
func toSlash(path string) string {
	return strings.ReplaceAll(path, string(os.PathSeparator), "/")
}

// trimSlash убирает начальные и конечные слэши из prefix
func trimSlash(prefix string) string {
	return strings.Trim(prefix, "/")
}
