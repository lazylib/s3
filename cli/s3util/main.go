package main

import (
	"flag"
	"fmt"

	"github.com/lazylib/s3"
)

func main() {
	folder := flag.String("folder", "./", "путь к директории которую нужно загрузить")
	url := flag.String("url", "", "Url s3")
	access := flag.String("access", "", "accesskey")
	secret := flag.String("secret", "", "secretkey")
	bucket := flag.String("bucket", "", "bucket")
	pathstyle := flag.Bool("pathstyle", true, "использовать pathstyle по умолчанию true")
	prefix := flag.String("prefix", "", "prefix для целевого пути")

	flag.Parse()

	// проверь аргументы и выведи красивую и понятную ошибку с предзаполнеными параметрами чтобы было проще скопировать и использовать

	conf := s3.New(s3.Settings{
		Url:          *url,
		AccessKey:    *access,
		Region:       "ru",
		Bucket:       *bucket,
		SecretKey:    *secret,
		UsePathStyle: *pathstyle,
	})

	// реализуй проход по всем файлам и загрузку через conf

	// key получи из суммы текущей пути до файла и prefix
	// filepath можно передать путь до файла
	// также покази процесс загрузки
	conf.Upload(key, filepath)

	fmt.Println(">>", *folder)

}
