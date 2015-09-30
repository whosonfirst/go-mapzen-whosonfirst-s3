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
	var success int64
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
		err := sink.SyncFile(source, dest)

		if err != nil {
			failed++
			return nil // don't stop believing
		}

		success++
		return nil
	}

	c := whosonfirst.NewCrawler(root)
	_ = c.Crawl(callback)

	t1 := float64(time.Since(t0)) / 1e9

	msg := fmt.Sprintf("processed %d files (ok: %d error: %d) in %.3f seconds\n", files, success, failed, t1)
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

	hash := md5.Sum(body)
	hex := enc.EncodeToString(hash[:])

	_, err = sink.Pool.SendWork(func() {

		sink.LogMessage(fmt.Sprintf("PUT %s as %s", dest, sink.ACL))

		headers := make(http.Header)
		rsp, _ := sink.Bucket.Head(dest, headers)

		etag := rsp.Header.Get("Etag")
		etag = strings.Replace(etag, "\"", "", -1)

		if etag == hex  {
		   sink.LogMessage(fmt.Sprintf("SKIP %s because it is unchanged", dest))
		   return
		} 

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
