package whosonfirst

// https://github.com/aws/aws-sdk-go
// https://docs.aws.amazon.com/sdk-for-go/api/service/s3.html

// https://github.com/goamz/goamz/blob/master/aws/aws.go
// https://github.com/goamz/goamz/blob/master/s3/s3.go

import (
        "crypto/md5"
	enc "encoding/hex"
	"fmt"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/jeffail/tunny"
	"github.com/whosonfirst/go-mapzen-whosonfirst-crawl/src/com.mapzen/whosonfirst"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

type Sync struct {
	ACL    s3.ACL
	Bucket s3.Bucket
	Pool   tunny.WorkPool
	Log    chan string
}

func New(auth aws.Auth, region aws.Region, acl s3.ACL, bucket string, log chan string) *Sync {

	numCPUs := runtime.NumCPU() * 2
	runtime.GOMAXPROCS(numCPUs)

	p, _ := tunny.CreatePoolGeneric(numCPUs).Open()

	s := s3.New(auth, region)
	b := s.Bucket(bucket)

	return &Sync{
		ACL:    acl,
		Bucket: *b,
		Pool:   *p,
		Log:    log,
	}
}

func WhosOnFirst(auth aws.Auth, bucket string, log chan string) *Sync {

	return New(auth, aws.USEast, s3.PublicRead, bucket, log)
}

func (sink Sync) SyncDirectory(root string) error {

	defer sink.Pool.Close()

	var files int64
	var failed int64

	t0 := time.Now()

	callback := func(path string, info os.FileInfo) error {

		// sink.LogMessage(fmt.Sprintf("crawling %s", path))

		if info.IsDir() {
			return nil
		}

		files++

		source := path
		dest := source

		dest = strings.Replace(dest, root, "", -1)

		// Note: both HasChanged and SyncFile will ioutil.ReadFile(source)
		// which is a potential waste of time and resource. Or maybe we just
		// don't care? (20150930/thisisaaronland)

		change, ch_err := sink.HasChanged(source, dest)

		if ch_err != nil {
		   sink.LogMessage(fmt.Sprintf("failed to determine whether %s had changed, because '%s'", source, ch_err))
		   change = true
		}

		if change {

			s_err := sink.SyncFile(source, dest)

			if s_err != nil {
			   sink.LogMessage(fmt.Sprintf("failed to PUT %s, because '%s'", dest, s_err))
			   failed++
			}
                }
		
		return nil
	}

	c := whosonfirst.NewCrawler(root)
	_ = c.Crawl(callback)

	t1 := float64(time.Since(t0)) / 1e9

	msg := fmt.Sprintf("processed %d files (error: %d) in %.3f seconds\n", files, failed, t1)
	sink.LogMessage(msg)

	return nil
}

func (sink Sync) SyncFile(source string, dest string) error {

	// sink.LogMessage(fmt.Sprintf("sync file %s", source))

	body, err := ioutil.ReadFile(source)

	if err != nil {
		sink.LogMessage("OMGWTF")
		return err
	}

	_, err = sink.Pool.SendWork(func() {
		
		sink.LogMessage(fmt.Sprintf("PUT %s as %s", dest, sink.ACL))

		o := s3.Options{}

		err := sink.Bucket.Put(dest, body, "text/plain", sink.ACL, o)

		if err != nil {
			sink.LogMessage(fmt.Sprintf("failed to PUT %s, because '%s'", dest, err))
		}

	})

	if err != nil {
		sink.LogMessage(fmt.Sprintf("failed to schedule %s for processing, because '%s'", source, err))
		return err
	}

	// sink.LogMessage(fmt.Sprintf("scheduled %s for processing", source))
	return nil
}

// the following appears to trigger a freak-out-and-die condition... sometimes
// I have no idea why... test under go 1.2.1, 1.4.3 and 1.5.1 / see also:
// https://github.com/whosonfirst/go-mapzen-whosonfirst-s3/issues/2
// (2015/thisisaaronland)

func (sink Sync) HasChanged(source string, dest string) (ch bool, err error) {

     change := true

	body, err := ioutil.ReadFile(source)

	if err != nil {
	   return change, err
	}

	hash := md5.Sum(body)
	local_hash := enc.EncodeToString(hash[:])

	headers := make(http.Header)
	rsp, err := sink.Bucket.Head(dest, headers)

	if err != nil {
	   return change, err
	}

	etag := rsp.Header.Get("Etag")
	remote_hash := strings.Replace(etag, "\"", "", -1)

	if local_hash == remote_hash {
	  change = false
	}

	// sink.LogMessage(fmt.Sprintf("local: %s remote: %s change: %v", local_hash, remote_hash. change))

	return change, nil
}

// clearly I need to internalize this a bit more
// (20150930/thisisaaronland)
// https://talks.golang.org/2012/waza.slide

func (sink Sync) LogMessage(msg string) {

	fmt.Println(msg)

	/*
	   logger := func(ch chan string, txt string){
	   	    ch <- txt
	   }

	   go logger(sink.Log, msg)
	*/
}
