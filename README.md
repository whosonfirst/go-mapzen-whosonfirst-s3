# go-whosonfirst-s3

Go package for working with Who's On First data and S3 buckets

## Important

* This package has been superseded by [whosonfirst/go-whosonfirst-blob](https://github.com/whosonfirst/go-whosonfirst-blob) and is no longer maintained.
* There is only local -> remote (S3) synchronization at the moment. See above.
* There is no retry (for failed uploads) code yet.

## Install

You will need to have both `Go` (specifically version [1.12](https://golang.org/dl/) or higher) and the `make` programs installed on your computer. Assuming you do just type:

```
make tools
```

All of this package's dependencies are bundled with the code in the `vendor` directory.

## Usage

_Please write me_

## Tools

### wof-s3-delete

Given an ID (say `1159324849`) this will recursively delete everything in `PREFIX/115/932/484/9`.

```
./bin/wof-s3-delete -h
Usage of ./bin/wof-s3-delete:
  -dryrun
    	Go through the motions but don't actually delete anything.
  -lambda-clients int
    	The number of concurrent Lambda functions to invoke. (default 10)
  -lambda-dsn string
    	A valid go-whosonfirst-aws DSN string for talking to Lambda.
  -lambda-func string
    	The name of the Lambda function to invoke.
  -lambda-invoke
    	Invoke this code as a Lambda function.
  -lambda-type string
    	A valid go-aws-sdk lambda.InvocationType string (default "RequestResponse")
  -s3-dsn string
    	A valid go-whosonfirst-aws DSN string for talking to S3.
  -stdin
    	Read IDs to delete from STDIN.
```

For example:

```
$> cat /usr/local/data/to-delete.csv | ./bin/wof-s3-delete -lambda-invoke -lambda-dsn 'region=us-west-2 credentials=session' -lambda-func DeleteMedia -dryrun -stdin
```

### wof-s3-sync

```
./bin/wof-s3-sync -h
Usage of ./bin/wof-s3-sync:
  -acl string
       A valid AWS S3 ACL string for permissions. (default "public-read")
  -bucket string
    	  The name of your S3 bucket. (default "data.whosonfirst.org")
  -credentials string
    	       What kind of AWS credentials to use for syncing data. (default "iam:")
  -dryrun
	Go through the motions but don't actually sync anything.
  -dsn string
       A valid go-whosonfirst-aws DSN string.
  -force
	Sync local files even if they haven't changed remotely.
  -mode string
    	The mode to use for reading local data. Valid modes are: directory,feature,feature-collection,files,geojson-ls,meta,path,repo,sqlite. (default "repo")
  -prefix string
    	  The prefix (or subdirectory) for syncing data (default "data")
  -rate-limit int
    	      The maximum number or concurrent processes. (default 100000)
  -region string
    	  The region your S3 bucket lives in. (default "us-east-1")
  -verbose
	Be chatty.
```

For example:

```
./bin/wof-s3-sync -rate-limit 100000 -dsn 'bucket=data.whosonfirst.org region=us-east-1 prefix=data credentials=iam:' -mode repo /usr/local/data/whosonfirst-data
2017/12/12 14:12:02 109820 indexed
2017/12/12 14:13:02 209831 indexed
2017/12/12 14:14:02 309822 indexed
2017/12/12 14:15:02 409789 indexed
2017/12/12 14:16:02 509838 indexed
2017/12/12 14:17:02 609817 indexed
2017/12/12 14:18:02 709817 indexed
2017/12/12 14:19:02 809843 indexed
2017/12/12 14:20:02 909810 indexed
2017/12/12 14:20:23 time to index 9m20.532420899s
2017/12/12 14:20:23 time to index 936153 documents : 9m20.532461673s
```

## See also

* https://github.com/whosonfirst/go-whosonfirst-aws
* https://github.com/whosonfirst/go-whosonfirst-index
