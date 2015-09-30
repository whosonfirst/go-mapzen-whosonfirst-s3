package whosonfirst

// https://github.com/aws/aws-sdk-go
// https://docs.aws.amazon.com/sdk-for-go/api/service/s3.html

// https://github.com/goamz/goamz/blob/master/aws/aws.go
// https://github.com/goamz/goamz/blob/master/s3/s3.go

import (
	"fmt"
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
	"github.com/whosonfirst/go-mapzen-whosonfirst-crawl/src/com.mapzen/whosonfirst"
	"io/ioutil"
	"os"
	"strings"
	"github.com/jeffail/tunny"
	"runtime"
)

type Sync struct {
	ACL    s3.ACL
	Bucket s3.Bucket
	Files  int64
	Ok     int64
	Error  int64
	Pool   tunny.WorkPool
}

func New(auth aws.Auth, region aws.Region, acl s3.ACL, bucket string) *Sync {

     	numCPUs := runtime.NumCPU()
        runtime.GOMAXPROCS(numCPUs)

	p, _ := tunny.CreatePoolGeneric(numCPUs).Open()
	defer p.Close()

	s := s3.New(auth, region)
	b := s.Bucket(bucket)

	return &Sync{
		ACL:    acl,
		Bucket: *b,
		Pool: *p,
		Files:  0,
		Ok:     0,
		Error:  0,
	}
}

func WhosOnFirst(auth aws.Auth, bucket string) *Sync {

	return New(auth, aws.USEast, s3.PublicRead, bucket)
}

func (sink Sync) SyncDirectory(root string) error {

	callback := func(path string, info os.FileInfo) error {

		if info.IsDir() {
			return nil
		}

		source := path
		dest := source

		dest = strings.Replace(dest, root, "", -1)

		err := sink.SyncFile(source, dest)

		if err != nil {
			return err
		}

		return nil
	}

	c := whosonfirst.NewCrawler(root)
	_ = c.Crawl(callback)

	return nil
}

func (sink Sync) SyncFile(source string, dest string) error {

	sink.Files++

	body, err := ioutil.ReadFile(source)

	if err != nil {
		sink.Error++
		return err
	}

	fmt.Printf("sync %s to %s\n", source, dest)

	_, err = sink.Pool.SendWork(func(){

		fmt.Printf("send to %s w/ %s", dest, sink.ACL)

		o := s3.Options{}

		err := sink.Bucket.Put(dest, body, "text/plain", sink.ACL, o)

		if err != nil {
		   sink.Error++
		   fmt.Printf("%s\n", err)
		}

	        sink.Ok++
        })

	return nil
}