package main

import (
	sync "com.mapzen/whosonfirst/s3"
	"flag"
	"fmt"
	"github.com/goamz/goamz/aws"
	"os"
)

func main() {

	var root = flag.String("root", "", "The directory to sync")
	var bucket = flag.String("bucket", "", "The S3 bucket to sync <root> to")
	var credentials = flag.String("credentials", "", "Your S3 credentials file")

	flag.Parse()

	if *root == "" {
		panic("missing root to sync")
	}

	_, err := os.Stat(*root)

	if os.IsNotExist(err) {
		panic("root does not exist")
	}

	if *bucket == "" {
		panic("missing bucket")
	}

	if *credentials != "" {
		os.Setenv("AWS_CREDENTIAL_FILE", *credentials)
	}

	auth, err := aws.SharedAuth()

	if err != nil {
		panic(err)
	}

	fmt.Printf("sync to %s%s\n", *root, *bucket)

	sink := sync.WhosOnFirst(auth, *bucket)
	err = sink.SyncDirectory(*root)

	fmt.Printf("files:%d okay:%d error:%d", sink.Files, sink.Ok, sink.Error)

}
