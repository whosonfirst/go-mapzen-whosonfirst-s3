deps:
	go get -u "github.com/whosonfirst/go-whosonfirst-crawl/whosonfirst"
	go get -u "github.com/goamz/goamz/aws"
	go get -u "github.com/goamz/goamz/s3"
	go get -u "github.com/jeffail/tunny"

sync: 
	go build -o bin/sync bin/sync.go

fmt:
	go fmt whosonfirst/s3/sync.go 
	go fmt bin/sync.go 
