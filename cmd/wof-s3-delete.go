package main

import (
	"context"
	"errors"
	"flag"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/whosonfirst/go-whosonfirst-aws/s3"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"log"
	"os"
	"strconv"
)

type DeleteOptions struct {
	DSN    string `json:"dsn"`
	Dryrun bool   `json:"dryrun"`
	ID     int64  `json:"id"`
}

func delete(ctx context.Context, opts DeleteOptions) error {

	if opts.DSN == "" {

		dsn, ok := os.LookupEnv("DSN")

		if !ok {
			return errors.New("Missing DSN")
		}

		opts.DSN = dsn
	}

	if opts.ID == 0 {
		return errors.New("Invalid ID")
	}

	cfg, err := s3.NewS3ConfigFromString(opts.DSN)

	if err != nil {
		return err
	}

	conn, err := s3.NewS3Connection(cfg)

	if err != nil {
		return err
	}

	// add hooks for alternative paths (fullname, etc.)

	path, err := uri.Id2Path(opts.ID)

	if err != nil {
		return err
	}

	if opts.Dryrun {
		log.Println("DELETE", path)
		return nil
	}

	return conn.DeleteRecursive(path)
}

func main() {

	do_invoke := flag.Bool("invoke", false, "...")
	dryrun := flag.Bool("dryrun", false, "...")

	dsn := flag.String("dsn", "", "...")

	flag.Parse()

	opts := DeleteOptions{
		DSN:    *dsn,
		Dryrun: *dryrun,
	}

	_, do_lambda := os.LookupEnv("LAMBDA")

	if do_lambda {
		lambda.Start(delete)
		os.Exit(0)
	}

	if *do_invoke {

		log.Fatal("please write me")

	} else {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		for _, str_id := range flag.Args() {

			id, err := strconv.ParseInt(str_id, 10, 64)

			if err != nil {
				log.Fatal(err)
			}

			opts.ID = id
			err = delete(ctx, opts)

			if err != nil {
				log.Fatal(err)
			}
		}

		os.Exit(0)
	}
}
