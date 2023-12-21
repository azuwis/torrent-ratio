torrent-ratio: main.go
	go build -ldflags '-s -w'

fmt:
	go fmt

run:
	go run main.go -v -addr 127.0.0.1:8089 -conf torrent-ratio.yaml -db torrent-ratio.db

update:
	go get -u ./
	go mod tidy
