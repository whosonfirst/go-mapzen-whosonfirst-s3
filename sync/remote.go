package sync

import (
	"bytes"
	"context"
	"fmt"
	"github.com/aaronland/go-aws-s3"
	"github.com/whosonfirst/go-whosonfirst-iterate/emitter"
	"github.com/whosonfirst/go-whosonfirst-log"
	"github.com/whosonfirst/go-whosonfirst-s3/throttle"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"io"
	"path/filepath"
)

type RemoteSyncOptions struct {
	Region      string
	Bucket      string
	Prefix      string
	Credentials string
	DSN         string
	ACL         string
	RateLimit   int
	Force       bool
	Dryrun      bool
	Verbose     bool
	Logger      *log.WOFLogger
}

type RemoteSync struct {
	Sync
	conn     *s3.S3Connection
	options  RemoteSyncOptions
	throttle throttle.Throttle
}

func NewRemoteSync(opts RemoteSyncOptions) (Sync, error) {

	dsn := opts.DSN

	if dsn == "" {
		dsn = fmt.Sprintf("bucket=%s prefix=%s region=%s credentials=%s", opts.Bucket, opts.Prefix, opts.Region, opts.Credentials)
	}

	conn, err := s3.NewS3ConnectionWithDSN(dsn)

	if err != nil {
		return nil, fmt.Errorf("Failed to create S3 connection, %w", err)
	}

	th, err := throttle.NewThrottledThrottle(opts.RateLimit)

	if err != nil {
		return nil, fmt.Errorf("Failed to create throttle, %w", err)
	}

	rs := RemoteSync{
		options:  opts,
		conn:     conn,
		throttle: th,
	}

	return &rs, nil
}

func (s *RemoteSync) SyncFunc() (emitter.EmitterCallbackFunc, error) {

	cb := func(ctx context.Context, fh io.ReadSeeker, args ...interface{}) error {

		select {

		case <-ctx.Done():
			return nil
		default:
			// pass
		}

		path, err := emitter.PathForContext(ctx)

		if err != nil {
			return fmt.Errorf("Failed to derive path for context, %w", err)
		}

		if path == emitter.STDIN {
			return fmt.Errorf("Can't sync STDIN")
		}

		is_wof, err := uri.IsWOFFile(path)

		if err != nil {
			return fmt.Errorf("Failed to determine whether %s is a WOF file, %w", path, err)
		}

		if !is_wof {
			return nil
		}

		err = s.throttle.RateLimit()

		if err != nil {
			return fmt.Errorf("Failed to rate limit, %w", err)
		}

		err = s.SyncFile(ctx, fh, path)

		if err != nil {
			return fmt.Errorf("Failed to sync '%s', %w", path, err)
		}

		return nil
	}

	return cb, nil
}

func (s *RemoteSync) SyncFile(ctx context.Context, fh io.Reader, source string) error {

	select {
	case <-ctx.Done():
		return nil
	default:
		// pass
	}

	id, err := uri.IdFromPath(source)

	if err != nil {
		return fmt.Errorf("Failed to derive ID from path '%s', %w", source, err)
	}

	rel_path, err := uri.Id2RelPath(id)

	if err != nil {
		return fmt.Errorf("Failed to generate relative path for %d, %w", id, err)
	}

	root := filepath.Dir(rel_path)
	fname := filepath.Base(source)
	dest := filepath.Join(root, fname)

	key := fmt.Sprintf("%s#ACL=%s", dest, s.options.ACL)

	s.options.Logger.Debug("CHECK %d (%s) AS '%s'", id, rel_path, key)

	if !s.options.Force {

		body, err := io.ReadAll(fh)

		if err != nil {
			return fmt.Errorf("Failed to read %s, %w", rel_path, err)
		}

		changed, err := s.conn.HasChanged(ctx, dest, body)

		if err != nil {
			return fmt.Errorf("Failed to determined whether '%s' has changed, %w", rel_path, err)
		}

		s.options.Logger.Status("Has %s changed: %t", dest, changed)

		if !changed {
			return nil
		}

		fh = bytes.NewReader(body)
	}

	s.options.Logger.Status("PUT '%s'", key)

	if s.options.Dryrun {
		s.options.Logger.Status("Running in dryrun mode, so not PUT-ing anything...")
		return nil
	}

	closer := io.NopCloser(fh)

	err = s.conn.Put(ctx, key, closer)

	// s3/utils.IsAWSErrorWithCode

	/*
		if err != nil {

			aws_err := err.(awserr.Error)

			if aws_err.Code() == "SlowDown" {

			}
		}
	*/

	if err != nil {
		return fmt.Errorf("Failed to PUT '%s', %w", key, err)
	}

	return nil
}
