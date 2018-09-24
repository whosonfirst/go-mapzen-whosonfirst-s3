package main

import (
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-index"
	"github.com/whosonfirst/go-whosonfirst-log"
	"github.com/whosonfirst/go-whosonfirst-s3/sync"
	"io"
	"os"
	"sync/atomic"
	"time"
)

func main() {

	var mode = flag.String("mode", "repo", "...")
	var region = flag.String("region", "us-east-1", "...")
	var bucket = flag.String("bucket", "whosonfirst.mapzen.com", "...")
	var prefix = flag.String("prefix", "", "...")
	var dsn = flag.String("dsn", "", "...")
	var acl = flag.String("acl", "public-read", "...")
	var credentials = flag.String("credentials", "default", "...")
	var ratelimit = flag.Int("rate-limit", 100000, "...")
	var dryrun = flag.Bool("dryrun", false, "...")
	var force = flag.Bool("force", false, "...")
	var verbose = flag.Bool("verbose", false, "...")

	flag.Parse()

	logger := log.SimpleWOFLogger()

	if *verbose {
		stdout := io.Writer(os.Stdout)
		logger.AddLogger(stdout, "status")
	}

	if *dsn == "" {
		*dsn = fmt.Sprintf("bucket=%s prefix=%s region=%s credentials=%s", *bucket, *prefix, *region, *credentials)
	}

	opts := sync.RemoteSyncOptions{
		DSN:       *dsn,
		ACL:       *acl,
		RateLimit: *ratelimit,
		Dryrun:    *dryrun,
		Force:     *force,
		Verbose:   *verbose,
		Logger:    logger,
	}

	sync, err := sync.NewRemoteSync(opts)

	if err != nil {
		logger.Fatal("Failed to create new sync because %s", err)
	}

	cb, err := sync.SyncFunc()

	if err != nil {
		logger.Fatal("Failed to create sync callback because %s", err)
	}

	idx, err := index.NewIndexer(*mode, cb)

	if err != nil {
		logger.Fatal("Failed to create indexer because %s", err)
	}

	done_ch := make(chan bool)

	go func() {

		for {
			select {
			case <-done_ch:
				break
			case <-time.After(1 * time.Minute):
				i := atomic.LoadInt64(&idx.Indexed) // please just make this part of go-whosonfirst-index
				logger.Status("%d indexed\n", i)
			}
		}
	}()

	t1 := time.Now()

	for _, path := range flag.Args() {

		ta := time.Now()

		err := idx.IndexPath(path)

		if err != nil {
			logger.Warning("Failed to index %s because %s", path, err)
			break
		}

		tb := time.Since(ta)
		logger.Status("time to index %s : %v\n", path, tb)
	}

	// this code doesn't exist and I am not sure how I want to deal
	// with it yet (20171212/thisisaaronland)

	/*

		if sync.HasRetries() {

			idx, err = index.NewIndexer("files", cb)

			if err != nil {
				log.Fatal(err)
			}

			for _, path := range sync.Retries() {

				idx.IndexPath(path)
			}
		}

	*/

	done_ch <- true

	t2 := time.Since(t1)
	i := atomic.LoadInt64(&idx.Indexed) // see above

	logger.Status("time to index %d documents : %v\n", i, t2)
}
