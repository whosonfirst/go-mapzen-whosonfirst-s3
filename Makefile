CWD=$(shell pwd)
GOPATH := $(CWD)

prep:
	if test -d pkg; then rm -rf pkg; fi

self:	prep rmdeps
	if test -d src/github.com/whosonfirst/go-whosonfirst-s3; then rm -rf src/github.com/whosonfirst/go-whosonfirst-s3; fi
	mkdir -p src/github.com/whosonfirst/go-whosonfirst-s3
	cp s3.go src/github.com/whosonfirst/go-whosonfirst-s3/
	cp -r sync src/github.com/whosonfirst/go-whosonfirst-s3/
	cp -r throttle src/github.com/whosonfirst/go-whosonfirst-s3/
	cp -r vendor/* src/

rmdeps:
	if test -d src; then rm -rf src; fi 

build:	fmt bin

deps: 	rmdeps
	@GOPATH=$(shell pwd) go get -u "github.com/hashicorp/golang-lru"
	@GOPATH=$(shell pwd) go get -u "github.com/throttled/throttled"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-index"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-log"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-uri"
	@GOPATH=$(shell pwd) go get -u "github.com/whosonfirst/go-whosonfirst-aws/..."

vendor-deps: rmdeps deps
	if test -d vendor; then rm -rf vendor; fi
	cp -r src vendor
	find vendor -name '.git' -print -type d -exec rm -rf {} +
	rm -rf src

bin:	self
	@GOPATH=$(GOPATH) go build -o bin/wof-s3-sync cmd/wof-s3-sync.go

fmt:
	go fmt *.go 
	go fmt cmd/*.go
	go fmt sync/*.go
	go fmt throttle/*.go
