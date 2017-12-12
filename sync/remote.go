package sync

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	_ "fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/whosonfirst/go-whosonfirst-index"
	"github.com/whosonfirst/go-whosonfirst-s3/throttle"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type RemoteSyncOptions struct {
	Region      string
	Bucket      string
	Prefix      string
	ACL         string
	Credentials string
	RateLimit   int
	Force       bool
	Dryrun      bool
	Verbose     bool
}

type RemoteSync struct {
	Sync
	service  *s3.S3
	options  RemoteSyncOptions
	throttle throttle.Throttle
}

func NewRemoteSync(opts RemoteSyncOptions) (Sync, error) {

	cfg := aws.NewConfig()
	cfg.WithRegion(opts.Region)

	if strings.HasPrefix(opts.Credentials, "env:") {

		creds := credentials.NewEnvCredentials()
		cfg.WithCredentials(creds)

	} else if strings.HasPrefix(opts.Credentials, "shared:") {

		details := strings.Split(opts.Credentials, ":")

		if len(details) != 3 {
			return nil, errors.New("Shared credentials need to be defined as 'shared:CREDENTIALS_FILE:PROFILE_NAME'")
		}

		creds := credentials.NewSharedCredentials(details[1], details[2])
		cfg.WithCredentials(creds)

	} else if strings.HasPrefix(opts.Credentials, "iam:") {

		// assume an IAM role suffient for doing whatever

	} else {

		whoami, err := user.Current()

		if err != nil {
			return nil, err
		}

		dotaws := filepath.Join(whoami.HomeDir, ".aws")
		creds_file := filepath.Join(dotaws, "credentials")

		profile := "default"

		if opts.Credentials != "" {
			profile = opts.Credentials
		}

		creds := credentials.NewSharedCredentials(creds_file, profile)
		cfg.WithCredentials(creds)
	}

	sess := session.New(cfg)
	service := s3.New(sess)

	th, err := throttle.NewThrottledThrottle(opts.RateLimit)

	if err != nil {
		return nil, err
	}

	rs := RemoteSync{
		options:  opts,
		service:  service,
		throttle: th,
	}

	return &rs, nil
}

func (s *RemoteSync) SyncFunc() (index.IndexerFunc, error) {

	f := func(fh io.Reader, ctx context.Context, args ...interface{}) error {

		select {

		case <-ctx.Done():
			return nil
		default:

			path, err := index.PathForContext(ctx)

			if err != nil {
				return err
			}

			if path == index.STDIN {
				return errors.New("Can't sync STDIN")
			}

			is_wof, err := uri.IsWOFFile(path)

			if err != nil {
				return err
			}

			if !is_wof {
				return nil
			}

			err = s.throttle.RateLimit()

			if err != nil {
				return err
			}

			err = s.SyncFile(path)

			if err != nil {
				return err
			}

			return nil
		}
	}

	return f, nil
}

func (s *RemoteSync) SyncFile(source string) error {

	id, err := uri.IdFromPath(source)

	if err != nil {
		return err
	}

	rel_path, err := uri.Id2RelPath(id)

	if err != nil {
		return err
	}

	root := filepath.Dir(rel_path)
	fname := filepath.Base(source)
	dest := filepath.Join(root, fname)

	if s.options.Prefix != "" {
		dest = filepath.Join(s.options.Prefix, dest)
	}

	if !s.options.Force {

		changed, err := s.HasChanged(source, dest)

		if err != nil {
			return err
		}

		if !changed {
			return nil
		}
	}

	body, err := ioutil.ReadFile(source)

	if err != nil {
		return err
	}

	content_type := "application/json"

	if s.options.Verbose {
		log.Printf("SYNC %s to s3://%s/%s\n", source, s.options.Bucket, dest)
	}

	params := &s3.PutObjectInput{
		Bucket:      aws.String(s.options.Bucket),
		Key:         aws.String(dest),
		Body:        bytes.NewReader(body),
		ContentType: aws.String(content_type),
		ACL:         aws.String(s.options.ACL),
	}

	if s.options.Dryrun {
		return nil
	}

	_, err = s.service.PutObject(params)

	if err != nil {

		aws_err := err.(awserr.Error)

		if aws_err.Code() == "SlowDown" {

		}
	}

	return err
}

func (s *RemoteSync) HasChanged(source string, dest string) (bool, error) {

	// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#HeadObjectInput
	// https://docs.aws.amazon.com/sdk-for-go/api/service/s3/#HeadObjectOutput

	params := &s3.HeadObjectInput{
		Bucket: aws.String(s.options.Bucket),
		Key:    aws.String(dest),
	}

	rsp, err := s.service.HeadObject(params)

	if err != nil {

		aws_err := err.(awserr.Error)

		if aws_err.Code() == "NotFound" {
			return true, nil
		}

		if aws_err.Code() == "SlowDown" {

		}

		return false, err
	}

	body, err := ioutil.ReadFile(source)

	if err != nil {
		return false, err
	}

	enc := md5.Sum(body)
	local_hash := hex.EncodeToString(enc[:])

	etag := *rsp.ETag
	remote_hash := strings.Replace(etag, "\"", "", -1)

	// log.Println("HASH", local_hash, remote_hash)

	if local_hash == remote_hash {
		return false, nil
	}

	// Okay so we think that things have changed but let's just check
	// modification times to be extra sure (20151112/thisisaaronland)

	info, err := os.Stat(source)

	if err != nil {
		return false, err
	}

	mtime_local := info.ModTime()
	mtime_remote := *rsp.LastModified

	// Because who remembers this stuff anyway...
	// func (t Time) Before(u Time) bool
	// Before reports whether the time instant t is before u.

	if mtime_local.Before(mtime_remote) {
		return false, nil
	}

	return true, nil
}
