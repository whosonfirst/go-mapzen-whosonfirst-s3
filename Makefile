prep:
	if test -d pkg; then rm -rf pkg; fi

self:	prep
	if test -d src/github.com/whosonfirst/go-whosonfirst-s3; then rm -rf src/github.com/whosonfirst/go-whosonfirst-s3; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-s3
	cp s3.go src/github.com/whosonfirst/go-whosonfirst-s3/

rmdeps:
	if test -d src; then rm -rf src; fi 

build:	rmdeps deps bin

deps: 	self
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-crawl"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-log"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-pool"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-utils"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-writer-slackcat"
	@GOPATH=$(shell pwd) go get -u "github.com/goamz/goamz/aws"
	@GOPATH=$(shell pwd) go get -u "github.com/goamz/goamz/s3"
	@GOPATH=$(shell pwd) go get -u "github.com/jeffail/tunny"

bin:	sync-dirs sync-files

sync-dirs: fmt self
	@GOPATH=$(shell pwd) go build -o bin/wof-sync-dirs cmd/wof-sync-dirs.go

sync-files: fmt self
	@GOPATH=$(shell pwd) go build -o bin/wof-sync-files cmd/wof-sync-files.go

fmt:
	go fmt *.go 
	go fmt cmd/*.go
