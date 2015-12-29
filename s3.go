package s3

// https://github.com/aws/aws-sdk-go
// https://docs.aws.amazon.com/sdk-for-go/api/service/s3.html

// https://github.com/goamz/goamz/blob/master/aws/aws.go
// https://github.com/goamz/goamz/blob/master/s3/s3.go

import (
	"bufio"
	"fmt"
	"github.com/goamz/goamz/aws"
	aws_s3 "github.com/goamz/goamz/s3"
	"github.com/jeffail/tunny"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	log "github.com/whosonfirst/go-whosonfirst-log"
	pool "github.com/whosonfirst/go-whosonfirst-pool"
	utils "github.com/whosonfirst/go-whosonfirst-utils"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Sync struct {
	ACL           aws_s3.ACL
	Bucket        aws_s3.Bucket
	Prefix        string
	WorkPool      tunny.WorkPool
	Logger        *log.WOFLogger
	Debug         bool
	Success       int64
	Error         int64
	Skipped       int64
	Scheduled     int64
	Completed     int64
	Retried       int64
	TimeToProcess *time.Duration
	Retries       *pool.LIFOPool
	MaxRetries    float64 // max percentage of errors over scheduled
}

func NewSync(auth aws.Auth, region aws.Region, acl aws_s3.ACL, bucket string, prefix string, procs int, debug bool, logger *log.WOFLogger) *Sync {

	logger.Info("creating a new Sync thing-y with %d processes", procs)

	runtime.GOMAXPROCS(procs)

	workpool, _ := tunny.CreatePoolGeneric(procs).Open()

	retries := pool.NewLIFOPool()

	s := aws_s3.New(auth, region)
	b := s.Bucket(bucket)

	ttp := new(time.Duration)

	return &Sync{
		ACL:           acl,
		Bucket:        *b,
		Prefix:        prefix,
		WorkPool:      *workpool,
		Debug:         debug,
		Logger:        logger,
		Scheduled:     0,
		Completed:     0,
		Skipped:       0,
		Error:         0,
		Success:       0,
		Retried:       0,
		TimeToProcess: ttp,
		Retries:       retries,
		MaxRetries:    25.0, // maybe allow this to be user-defined ?
	}
}

func WOFSync(auth aws.Auth, bucket string, prefix string, procs int, debug bool, logger *log.WOFLogger) *Sync {

	return NewSync(auth, aws.USEast, aws_s3.PublicRead, bucket, prefix, procs, debug, logger)
}

func (sink *Sync) SyncDirectory(root string) error {

	defer sink.WorkPool.Close()

	t0 := time.Now()

	wg := new(sync.WaitGroup)

	callback := func(source string, info os.FileInfo) error {

		wg.Add(1)

		if info.IsDir() {
			return nil
		}

		err := sink.SyncFile(source, root, wg)

		if err != nil {
			sink.Logger.Error("failed to sync %s, because '%s'", source, err)
		}

		return nil
	}

	c := crawl.NewCrawler(root)
	_ = c.Crawl(callback)

	wg.Wait()

	sink.ProcessRetries(root)

	ttp := time.Since(t0)
	sink.TimeToProcess = &ttp

	return nil
}

func (sink *Sync) SyncFiles(files []string, root string) error {

	defer sink.WorkPool.Close()

	t0 := time.Now()

	wg := new(sync.WaitGroup)

	for _, path := range files {

		wg.Add(1)

		go func(path string, root string, wg *sync.WaitGroup) {
			sink.SyncFile(path, root, wg)
		}(path, root, wg)
	}

	wg.Wait()

	sink.ProcessRetries(root)

	ttp := time.Since(t0)
	sink.TimeToProcess = &ttp

	return nil
}

func (sink *Sync) SyncFileList(path string, root string) error {

	defer sink.WorkPool.Close()

	t0 := time.Now()

	file, err := os.Open(path)

	if err != nil {
		return err
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)

	wg := new(sync.WaitGroup)

	for scanner.Scan() {

		path := scanner.Text()

		wg.Add(1)

		go func(path string, root string, wg *sync.WaitGroup) {
			sink.SyncFile(path, root, wg)
		}(path, root, wg)
	}

	wg.Wait()

	sink.ProcessRetries(root)

	ttp := time.Since(t0)
	sink.TimeToProcess = &ttp

	return nil
}

func (sink *Sync) SyncFile(source string, root string, wg *sync.WaitGroup) error {

	atomic.AddInt64(&sink.Scheduled, 1)

	_, err := sink.WorkPool.SendWork(func() {

		defer wg.Done()

		dest := source

		dest = strings.Replace(dest, root, "", -1)

		if sink.Prefix != "" {
			dest = path.Join(sink.Prefix, dest)
		}

		// Note: both HasChanged and SyncFile will ioutil.ReadFile(source)
		// which is a potential waste of time and resource. Or maybe we just
		// don't care? (20150930/thisisaaronland)

		sink.Logger.Debug("Looking for changes to %s (%s)", dest, sink.Prefix)

		change, ch_err := sink.HasChanged(source, dest)

		if ch_err != nil {
			atomic.AddInt64(&sink.Completed, 1)
			atomic.AddInt64(&sink.Error, 1)
			sink.Logger.Warning("failed to determine whether %s had changed, because '%s'", source, ch_err)

			sink.Retries.Push(&pool.PoolString{String: source})
			return
		}

		if sink.Debug == true {
			atomic.AddInt64(&sink.Completed, 1)
			atomic.AddInt64(&sink.Skipped, 1)
			sink.Logger.Debug("has %s changed? the answer is %v but does it really matter since debugging is enabled?", source, change)
			return
		}

		if !change {
			atomic.AddInt64(&sink.Completed, 1)
			atomic.AddInt64(&sink.Skipped, 1)
			sink.Logger.Debug("%s has not changed, skipping", source)
			return
		}

		err := sink.DoSyncFile(source, dest)

		if err != nil {
			sink.Retries.Push(&pool.PoolString{String: source})
			atomic.AddInt64(&sink.Error, 1)
		} else {
			atomic.AddInt64(&sink.Success, 1)
		}

		atomic.AddInt64(&sink.Completed, 1)
	})

	if err != nil {
		wg.Done()
		atomic.AddInt64(&sink.Error, 1)
		sink.Logger.Error("failed to schedule %s for processing, because %v", source, err)
		return err
	}

	sink.Logger.Debug("schedule %s for processing", source)
	return nil
}

func (sink *Sync) DoSyncFile(source string, dest string) error {

	sink.Logger.Debug("prepare %s for syncing", source)

	body, err := ioutil.ReadFile(source)

	if err != nil {
		sink.Logger.Error("Failed to read %s, because %v", source, err)
		return err
	}

	sink.Logger.Debug("PUT %s as %s", dest, sink.ACL)

	o := aws_s3.Options{}

	err = sink.Bucket.Put(dest, body, "text/plain", sink.ACL, o)

	if err != nil {
		sink.Logger.Error("failed to PUT %s, because '%s'", dest, err)
		return err
	}

	return nil
}

func (sink *Sync) HasChanged(source string, dest string) (ch bool, err error) {

	headers := make(http.Header)
	rsp, err := sink.Bucket.Head(dest, headers)

	if err != nil {

		if e, ok := err.(*aws_s3.Error); ok && e.StatusCode == 404 {
			sink.Logger.Debug("%s is 404 so assuming it has changed (WHOA)", dest)
			return true, nil
		}

		sink.Logger.Error("failed to HEAD %s because %s", dest, err)
		return false, err
	}

	local_hash, err := utils.HashFile(source)

	if err != nil {
		sink.Logger.Warning("failed to hash %s, because %v", source, err)
		return false, err
	}

	etag := rsp.Header.Get("Etag")
	remote_hash := strings.Replace(etag, "\"", "", -1)

	if local_hash == remote_hash {
		return false, nil
	}

	// Okay so we think that things have changed but let's just check
	// modification times to be extra sure (20151112/thisisaaronland)

	info, err := os.Stat(source)

	if err != nil {
		sink.Logger.Error("failed to stat %s because %s", source, err)
		return false, err
	}

	mtime_local := info.ModTime()

	last_mod := rsp.Header.Get("Last-Modified")
	mtime_remote, err := time.Parse(time.RFC1123, last_mod)

	if err != nil {
		sink.Logger.Error("failed to parse timestamp %s because %s", last_mod, err)
		return false, err
	}

	// Because who remembers this stuff anyway...
	// func (t Time) Before(u Time) bool
	// Before reports whether the time instant t is before u.

	sink.Logger.Debug("local %s %s", mtime_local, source)
	sink.Logger.Debug("remote %s %s", mtime_remote, dest)

	if mtime_local.Before(mtime_remote) {
		sink.Logger.Warning("remote copy of %s has a more recent modification date (local: %s remote: %s)", source, mtime_local, mtime_remote)
		return false, nil
	}

	return true, nil
}

func (sink *Sync) ProcessRetries(root string) bool {

	to_retry := sink.Retries.Length()

	if to_retry > 0 {

		scheduled_f := float64(sink.Scheduled)
		retry_f := float64(to_retry)

		pct := (retry_f / scheduled_f) * 100.0

		if pct > sink.MaxRetries {
			sink.Logger.Warning("E_EXCESSIVE_ERRORS, %f percent of scheduled processes failed thus undermining our faith that they will work now...", pct)
			return false
		}

		sink.Logger.Info("There are %d failed requests that will now be retried", to_retry)

		wg := new(sync.WaitGroup)

		for sink.Retries.Length() > 0 {

			r, ok := sink.Retries.Pop()

			if !ok {
				sink.Logger.Error("failed to pop retries because... computers?")
				break
			}

			source := r.StringValue()

			wg.Add(1)

			go func(source string, root string, wg *sync.WaitGroup) {

				atomic.AddInt64(&sink.Scheduled, 1)

				sink.WorkPool.SendWork(func() {

					atomic.AddInt64(&sink.Retried, 1)

					sink.Logger.Info("retry syncing %s", source)

					sink.SyncFile(source, root, wg)

					atomic.AddInt64(&sink.Completed, 1)
				})

			}(source, root, wg)
		}

		wg.Wait()
	}

	return true
}

func (sink *Sync) MonitorStatus() {

	go func() {

		t0 := time.Now()

		for {

			rpt := sink.StatusReport()
			ttp := time.Since(t0)

			sink.Logger.Info("%s Time %v", rpt, ttp)

			time.Sleep(10 * time.Second)

			if sink.Scheduled == sink.Completed {
				break
			}
		}

		sink.Logger.Info(sink.StatusReport())
		sink.Logger.Info("monitoring complete")
	}()
}

func (sink *Sync) StatusReport() string {
	return fmt.Sprintf("Scheduled %d Completed %d Success %d Error %d Skipped %d Retried %d Goroutines %d",
		sink.Scheduled, sink.Completed, sink.Success, sink.Error, sink.Skipped, sink.Retried, runtime.NumGoroutine())
}
