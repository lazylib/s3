include .env

upload:
	go run . -url "${S3_TARGET}" -access "${S3_KEY}" -secret "${S3_SECRET}" -bucket "${BUCKET}" -folder "./data"