deps:
	go get "github.com/whosonfirst/go-mapzen-whosonfirst-crawl/"
	go get "github.com/goamz/goamz"

sync: 
	go build -o bin/sync bin/sync.go

fmt:
	go fmt src/com.mapzen/whosonfirst/s3/sync.go 
	go fmt bin/sync.go 
