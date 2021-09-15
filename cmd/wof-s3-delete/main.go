package main

/*

Given an ID (1159324849) this will recursively delete everything
in PREFIX/115/932/484/9 - that is all (20181226/thisisaaronland)

*/

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"github.com/aaronland/go-aws-lambda"
	"github.com/aaronland/go-aws-s3"
	go_lambda "github.com/aws/aws-lambda-go/lambda"
	"github.com/whosonfirst/go-whosonfirst-uri"
	"log"
	"os"
	"strconv"
	"sync"
)

type DeleteOptions struct {
	DSN    string `json:"dsn"`
	Dryrun bool   `json:"dryrun"`
	ID     int64  `json:"id"`
}

func append_id(ids []int64, str_id string) ([]int64, error) {

	id, err := strconv.ParseInt(str_id, 10, 64)

	if err != nil {
		return ids, err
	}

	ids = append(ids, id)
	return ids, nil
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

	conn, err := s3.NewS3ConnectionWithDSN(opts.DSN)

	// add hooks for alternative paths (fullname, etc.)

	path, err := uri.Id2Path(opts.ID)

	if err != nil {
		return err
	}

	if opts.Dryrun {
		log.Println("[dryrun] DELETE", path)
		return nil
	}

	return conn.DeleteRecursive(ctx, path)
}

func main() {

	dryrun := flag.Bool("dryrun", false, "Go through the motions but don't actually delete anything.")
	stdin := flag.Bool("stdin", false, "Read IDs to delete from STDIN.")

	s3_dsn := flag.String("s3-dsn", "", "A valid go-whosonfirst-aws DSN string for talking to S3.")

	do_invoke := flag.Bool("lambda-invoke", false, "Invoke this code as a Lambda function.")
	lambda_dsn := flag.String("lambda-dsn", "", "A valid go-whosonfirst-aws DSN string for talking to Lambda.")
	lambda_func := flag.String("lambda-func", "", "The name of the Lambda function to invoke.")
	lambda_clients := flag.Int("lambda-clients", 10, "The number of concurrent Lambda functions to invoke.")
	lambda_type := flag.String("lambda-type", "RequestResponse", "A valid go-aws-sdk lambda.InvocationType string")

	// do_sqs := flag.Bool("sqs-invoke", false, "")
	// sqs_dsn := flag.String("sqs-dsn", "", "...")

	flag.Parse()

	ctx := context.Background()

	opts := DeleteOptions{
		DSN:    *s3_dsn,
		Dryrun: *dryrun,
	}

	_, do_lambda := os.LookupEnv("LAMBDA")

	if do_lambda {
		go_lambda.Start(delete)
		os.Exit(0)
	}

	ids := make([]int64, 0)
	var err error

	if *stdin {

		scanner := bufio.NewScanner(os.Stdin)

		for scanner.Scan() {

			ids, err = append_id(ids, scanner.Text())

			if err != nil {
				log.Fatal(err)
			}
		}

	} else {

		for _, str_id := range flag.Args() {

			ids, err = append_id(ids, str_id)

			if err != nil {
				log.Fatal(err)
			}
		}
	}

	/*

		for example:

		$> cat /usr/local/data/to-delete.csv | ./bin/wof-s3-delete -lambda-invoke -lambda-dsn 'region=us-west-2 credentials=session' -lambda-func DeleteMedia -dryrun -stdin
		2018/12/26 12:15:23 INVOKE 1159337327 200
		... and so on

	*/

	if *do_invoke {

		lambda_fn, err := lambda.NewLambdaFunctionWithDSN(*lambda_dsn, *lambda_func, *lambda_type)

		if err != nil {
			log.Fatal(err)
		}

		wg := new(sync.WaitGroup)

		throttle := make(chan bool, *lambda_clients)

		for i := 0; i < *lambda_clients; i++ {
			throttle <- true
		}

		for _, id := range ids {

			wg.Add(1)

			go func(lambda_fn *lambda.LambdaFunction, wg *sync.WaitGroup, throttle chan bool, id int64) {

				<-throttle

				defer func() {
					throttle <- true
					wg.Done()
				}()

				opts.ID = id

				_, err := lambda_fn.Invoke(ctx, id)

				if err != nil {
					log.Println("ERROR", id, err)
				}

			}(lambda_fn, wg, throttle, id)
		}

		wg.Wait()
		os.Exit(0)
	}

	// nothing left but the command line

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, id := range ids {

		opts.ID = id
		err := delete(ctx, opts)

		if err != nil {
			log.Fatal(err)
		}
	}

	os.Exit(0)
}
