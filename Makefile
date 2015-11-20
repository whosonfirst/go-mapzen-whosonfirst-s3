prep:
	if test -d pkg; then rm -rf pkg; fi

self:	prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-s3; then rm -rf src/github.com/whosonfirst/go-whosonfirst-s3; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-s3
	cp s3.go src/github.com/whosonfirst/go-whosonfirst-s3/

deps: 	self
	go get -u "github.com/whosonfirst/go-whosonfirst-crawl"
	go get -u "github.com/whosonfirst/go-whosonfirst-log"
	go get -u "github.com/whosonfirst/go-whosonfirst-pool"
	go get -u "github.com/whosonfirst/go-whosonfirst-utils"
	go get -u "github.com/whosonfirst/go-slackcat-writer"
	go get -u "github.com/goamz/goamz/aws"
	go get -u "github.com/goamz/goamz/s3"
	go get -u "github.com/jeffail/tunny"

bin:	sync-dirs sync-files

sync-dirs: fmt self
	go build -o bin/wof-sync-dirs cmd/wof-sync-dirs.go

sync-files: fmt self
	go build -o bin/wof-sync-files cmd/wof-sync-files.go

fmt:
	go fmt *.go 
	go fmt cmd/*.go
