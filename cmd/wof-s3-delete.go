package main

import (
	"context"
	"flag"
	"github.com/whosonfirst/go-whosonfirst-aws/s3"
	"github.com/whosonfirst/go-whosonfirst-cli/flags"
	"github.com/aws/aws-lambda-go/lambda"	
	"log"
	"os"
)

type DeleteOptions struct {
	DSN string `json:"dsn"`
	Path string `json:"path"`
	Recursive bool `json:"recursive"`
}

func delete(ctx context.Context, opts DeleteOptions) error {

	cfg, err := s3.NewS3ConfigFromString(opts.DSN)

	if err != nil {
		return err
	}
	
	conn, err := s3.NewS3Connection(cfg)

	if err != nil {
		return err
	}

	if opts.Recursive {
		return conn.DeleteRecursive(opts.Path)
	} else {
		return conn.Delete(opts.Path)
	}
}

func main(){

	do_lambda := flag.Bool("lambda", false, "...")
	do_invoke := flag.Bool("invoke", false, "...")

	dsn := flag.String("dsn", "", "...")
	path := flag.String("path", "", "...")
	recursive := flag.Bool("recursive", true, "...")		
	
	flag.Parse()

	flags.SetFlagsFromEnvVars("WOF_DELETE")
	
	if *path == "" {
		log.Fatal("Missing path")
	}
	
	opts := DeleteOptions{
		DSN: *dsn,
		Path: *path,
		Recursive: *recursive,
	}

	if *do_lambda {
		lambda.Start(delete)
		
	} else if *do_invoke {	

		log.Fatal("please write me")
		
	} else {
		
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := delete(ctx, opts)

		if err != nil {
			log.Fatal(err)
		}

		os.Exit(0)
	}
}
