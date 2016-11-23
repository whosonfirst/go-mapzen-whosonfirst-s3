prep:
	if test -d pkg; then rm -rf pkg; fi

self:	prep rmdeps
	if test -d src/github.com/whosonfirst/go-whosonfirst-s3; then rm -rf src/github.com/whosonfirst/go-whosonfirst-s3; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-s3
	cp s3.go src/github.com/whosonfirst/go-whosonfirst-s3/
	cp -r vendor/src/* src/

rmdeps:
	if test -d src; then rm -rf src; fi 

build:	fmt bin

deps: 	rmdeps
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-crawl"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-log"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-pool"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-utils"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-writer-slackcat"
	@GOPATH=$(shell pwd) go get -u "github.com/goamz/goamz/aws"
	@GOPATH=$(shell pwd) go get -u "github.com/goamz/goamz/s3"
	@GOPATH=$(shell pwd) go get -u "github.com/jeffail/tunny"

vendor-deps: deps
	if test ! -d vendor; then mkdir vendor; fi
	if test -d vendor/src; then rm -rf vendor/src; fi
	cp -r src vendor/src
	find vendor -name '.git' -print -type d -exec rm -rf {} +
	rm -rf src

bin:	sync-dirs sync-files

sync-dirs: fmt self
	@GOPATH=$(shell pwd) go build -o bin/wof-sync-dirs cmd/wof-sync-dirs.go

sync-files: fmt self
	@GOPATH=$(shell pwd) go build -o bin/wof-sync-files cmd/wof-sync-files.go

fmt:
	go fmt *.go 
	go fmt cmd/*.go
