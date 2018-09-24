package sync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/whosonfirst/go-whosonfirst-aws/s3"
	"github.com/whosonfirst/go-whosonfirst-index"
	"github.com/whosonfirst/go-whosonfirst-s3/throttle"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"io"
	"io/ioutil"
	"log"
	"path/filepath"
)

type RemoteSyncOptions struct {
	Region      string
	Bucket      string
	Prefix      string
	Credentials string
	ACL         string
	RateLimit   int
	Force       bool
	Dryrun      bool
	Verbose     bool
}

type RemoteSync struct {
	Sync
	config   *s3.S3Config
	conn     *s3.S3Connection
	options  RemoteSyncOptions
	throttle throttle.Throttle
}

func NewRemoteSync(opts RemoteSyncOptions) (Sync, error) {

	dsn := fmt.Sprintf("bucket=%s prefix=%s region=%s credentials=%s", opts.Bucket, opts.Prefix, opts.Region, opts.Credentials)
	log.Println("DSN", dsn)

	cfg, err := s3.NewS3ConfigFromString(dsn)

	if err != nil {
		return nil, err
	}

	conn, err := s3.NewS3Connection(cfg)

	if err != nil {
		return nil, err
	}

	th, err := throttle.NewThrottledThrottle(opts.RateLimit)

	if err != nil {
		return nil, err
	}

	rs := RemoteSync{
		options:  opts,
		config:   cfg,
		conn:     conn,
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
			// pass
		}

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

		err = s.SyncFile(fh, path)

		if err != nil {
			return err
		}

		return nil
	}

	return f, nil
}

func (s *RemoteSync) SyncFile(fh io.Reader, source string) error {

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

	if !s.options.Force {

		body, err := ioutil.ReadAll(fh)

		if err != nil {
			return err
		}

		changed, err := s.conn.HasChanged(dest, body)

		log.Println("CHANGED", dest, changed)

		if err != nil {
			return err
		}

		if !changed {
			return nil
		}

		fh = bytes.NewReader(body)
	}

	key := fmt.Sprintf("%s#ACL=%s", dest, s.options.ACL)

	log.Println("SYNC", key)
	return nil

	if s.options.Dryrun {
		return nil
	}

	closer := ioutil.NopCloser(fh)

	err = s.conn.Put(key, closer)

	// s3/utils.IsAWSErrorWithCode

	/*
		if err != nil {

			aws_err := err.(awserr.Error)

			if aws_err.Code() == "SlowDown" {

			}
		}
	*/

	return err
}
