torrent-ratio: main.go
	go build -ldflags '-s -w'

fmt:
	go fmt

run:
	go run main.go -v -addr :8089

update:
	go get -u ./
	go mod tidy
