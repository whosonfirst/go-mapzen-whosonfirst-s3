package main

import (
	"flag"
	"github.com/goamz/goamz/aws"
	log "github.com/whosonfirst/go-whosonfirst-log"
	"github.com/whosonfirst/go-whosonfirst-s3"
	"io"
	"os"
	"runtime"
)

func main() {

	var root = flag.String("root", "", "The directory PLEASE WRITE ME")
	var bucket = flag.String("bucket", "", "The S3 bucket to sync <root> to")
	var prefix = flag.String("prefix", "", "A prefix inside your S3 bucket where things go")
	var debug = flag.Bool("debug", false, "Don't actually try to sync anything and spew a lot of line noise")
	var credentials = flag.String("credentials", "", "Your S3 credentials file")
	var procs = flag.Int("processes", (runtime.NumCPU() * 2), "The number of concurrent processes to sync data with")
	var loglevel = flag.String("loglevel", "info", "Log level for reporting")

	flag.Parse()

	// read paths to sync from a file?
	args := flag.Args()

	if *root == "" {
		panic("missing root")
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

	writer := io.MultiWriter(os.Stdout)
	logger := log.NewWOFLogger(writer, "[wof-sync] ", *loglevel)

	s := s3.WOFSync(auth, *bucket, *prefix, *procs, *debug, logger)
	s.SyncFiles(args, *root)
}
