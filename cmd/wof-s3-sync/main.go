package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-iterate/iterator"
	"github.com/whosonfirst/go-whosonfirst-log"
	"github.com/whosonfirst/go-whosonfirst-s3/sync"
	"io"
	"os"
	"sync/atomic"
	"time"
)

func main() {

	var mode = flag.String("mode", "repo://", "...")
	var region = flag.String("region", "us-east-1", "The region your S3 bucket lives in.")
	var bucket = flag.String("bucket", "data.whosonfirst.org", "The name of your S3 bucket.")
	var prefix = flag.String("prefix", "", "The prefix (or subdirectory) for syncing data")
	var credentials = flag.String("credentials", "iam:", "What kind of AWS credentials to use for syncing data.")
	var dsn = flag.String("dsn", "", "A valid aaronland/go-aws-s3 DSN string.")
	var acl = flag.String("acl", "public-read", "A valid AWS S3 ACL string for permissions.")
	var ratelimit = flag.Int("rate-limit", 100000, "The maximum number or concurrent processes.")
	var dryrun = flag.Bool("dryrun", false, "Go through the motions but don't actually sync anything.")
	var force = flag.Bool("force", false, "Sync local files even if they haven't changed remotely.")
	var verbose = flag.Bool("verbose", false, "Be chatty.")

	flag.Parse()

	ctx := context.Background()

	logger := log.SimpleWOFLogger()

	if *verbose {
		stdout := io.Writer(os.Stdout)
		logger.AddLogger(stdout, "status")
	}

	if *dsn == "" {
		*dsn = fmt.Sprintf("bucket=%s prefix=%s region=%s credentials=%s", *bucket, *prefix, *region, *credentials)
	}

	logger.Status("DSN is %s", *dsn)

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

	iter_cb, err := sync.SyncFunc()

	if err != nil {
		logger.Fatal("Failed to create sync callback because %s", err)
	}

	iter, err := iterator.NewIterator(ctx, *mode, iter_cb)

	if err != nil {
		logger.Fatal("Failed to create iterator because %s", err)
	}

	done_ch := make(chan bool)

	go func() {

		for {
			select {
			case <-done_ch:
				break
			case <-time.After(1 * time.Minute):
				i := atomic.LoadInt64(&iter.Seen) // please just make this part of go-whosonfirst-index
				logger.Status("%d indexed\n", i)
			}
		}
	}()

	t1 := time.Now()

	uris := flag.Args()
	err = iter.IterateURIs(ctx, uris...)

	// please handle retries here

	done_ch <- true

	t2 := time.Since(t1)
	i := atomic.LoadInt64(&iter.Seen) // see above

	logger.Status("time to index %d documents : %v\n", i, t2)
}
