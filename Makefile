prep:
	if test -d pkg; then rm -rf pkg; fi

self:	prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-s3; then rm -rf src/github.com/whosonfirst/go-whosonfirst-s3; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-s3
	cp -r whosonfirst src/github.com/whosonfirst/go-whosonfirst-s3/whosonfirst

deps: 	self
	go get -u "github.com/whosonfirst/go-whosonfirst-crawl"
	go get -u "github.com/goamz/goamz/aws"
	go get -u "github.com/goamz/goamz/s3"
	go get -u "github.com/jeffail/tunny"

prep:	self
	rm -rf ./pkg/*

sync: 	prep
	go build -o bin/wof-sync bin/wof-sync.go

fmt:
	go fmt whosonfirst/s3/*.go 
	go fmt bin/*.go
