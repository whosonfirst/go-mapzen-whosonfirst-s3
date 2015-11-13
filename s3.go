package s3

// https://github.com/aws/aws-sdk-go
// https://docs.aws.amazon.com/sdk-for-go/api/service/s3.html

// https://github.com/goamz/goamz/blob/master/aws/aws.go
// https://github.com/goamz/goamz/blob/master/s3/s3.go

import (
	"bufio"
	"github.com/goamz/goamz/aws"
	aws_s3 "github.com/goamz/goamz/s3"
	"github.com/jeffail/tunny"
	"github.com/whosonfirst/go-whosonfirst-crawl"
	log "github.com/whosonfirst/go-whosonfirst-log"
	utils "github.com/whosonfirst/go-whosonfirst-utils"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"
)

type Sync struct {
	ACL    aws_s3.ACL
	Bucket aws_s3.Bucket
	Prefix string
	Pool   tunny.WorkPool
	Logger *log.WOFLogger
	Debug  bool
}

func NewSync(auth aws.Auth, region aws.Region, acl aws_s3.ACL, bucket string, prefix string, procs int, debug bool, logger *log.WOFLogger) *Sync {

	runtime.GOMAXPROCS(procs)

	pool, _ := tunny.CreatePoolGeneric(procs).Open()

	s := aws_s3.New(auth, region)
	b := s.Bucket(bucket)

	return &Sync{
		ACL:    acl,
		Bucket: *b,
		Prefix: prefix,
		Pool:   *pool,
		Debug:  debug,
		Logger: logger,
	}
}

func WOFSync(auth aws.Auth, bucket string, prefix string, procs int, debug bool, logger *log.WOFLogger) *Sync {

	return NewSync(auth, aws.USEast, aws_s3.PublicRead, bucket, prefix, procs, debug, logger)
}

func (sink Sync) SyncDirectory(root string) error {

	defer sink.Pool.Close()

	var files int64
	var failed int64

	t0 := time.Now()

	callback := func(source string, info os.FileInfo) error {

		if info.IsDir() {
			return nil
		}

		files++

		err := sink.SyncFile(source, root)

		if err != nil {
			sink.Logger.Error("failed to sync %s, because '%s'", source, err)
			failed++
		}

		return nil
	}

	c := crawl.NewCrawler(root)
	_ = c.Crawl(callback)

	t1 := float64(time.Since(t0)) / 1e9

	sink.Logger.Info("processed %d files (error: %d) in %.3f seconds\n", files, failed, t1)

	return nil
}

func (sink Sync) SyncFiles(files []string, root string) error {

	wg := new(sync.WaitGroup)

	for _, path := range files {

		wg.Add(1)

		go func() {
			defer wg.Done()
			sink.SyncFile(path, root)
		}()

	}

	wg.Wait()

	return nil
}

func (sink Sync) SyncFileList(path string, root string) error {

	file, err := os.Open(path)

	if err != nil {
		return err
	}

	defer file.Close()

	wg := new(sync.WaitGroup)

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {

		path := scanner.Text()

		wg.Add(1)

		go func() {
			defer wg.Done()
			sink.SyncFile(path, root)
		}()

	}

	wg.Wait()

	return nil
}

func (sink Sync) SyncFile(source string, root string) error {

	dest := source

	dest = strings.Replace(dest, root, "", -1)

	if sink.Prefix != "" {
		dest = path.Join(sink.Prefix, dest)
	}

	// Note: both HasChanged and SyncFile will ioutil.ReadFile(source)
	// which is a potential waste of time and resource. Or maybe we just
	// don't care? (20150930/thisisaaronland)

	sink.Logger.Debug("Looking for changes %s (%s)", dest, sink.Prefix)

	change, ch_err := sink.HasChanged(source, dest)

	if ch_err != nil {
		sink.Logger.Warning("failed to determine whether %s had changed, because '%s'", source, ch_err)
		change = true
	}

	if sink.Debug == true {
		sink.Logger.Debug("has %s changed? the answer is %v but does it really matter since debugging is enabled?", source, change)
		return nil
	}

	if !change {
		sink.Logger.Debug("%s has not changed, skipping", source)
		return nil
	}

	return sink.DoSyncFile(source, dest)
}

func (sink Sync) DoSyncFile(source string, dest string) error {

	sink.Logger.Debug("prepare %s for syncing", source)

	body, err := ioutil.ReadFile(source)

	if err != nil {
		sink.Logger.Error("Failed to read %s, because %v", source, err)
		return err
	}

	_, err = sink.Pool.SendWork(func() {

		sink.Logger.Debug("PUT %s as %s", dest, sink.ACL)

		o := aws_s3.Options{}

		err := sink.Bucket.Put(dest, body, "text/plain", sink.ACL, o)

		if err != nil {
			sink.Logger.Error("failed to PUT %s, because '%s'", dest, err)
		}

	})

	if err != nil {
		sink.Logger.Error("failed to schedule %s for processing, because '%s'", source, err)
		return err
	}

	sink.Logger.Debug("scheduled %s for processing", source)
	return nil
}

func (sink Sync) HasChanged(source string, dest string) (ch bool, err error) {

	headers := make(http.Header)
	rsp, err := sink.Bucket.Head(dest, headers)

	if err != nil {
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
