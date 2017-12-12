# go-mapzen-whosonfirst-s3

Go package for working with Who's On First data and S3 buckets

## Install

You will need to have both `Go` (specifically a version of Go more recent than 1.7 so let's just assume you need [Go 1.9](https://golang.org/dl/) or higher) and the `make` programs installed on your computer. Assuming you do just type:

```
make bin
```

All of this package's dependencies are bundled with the code in the `vendor` directory.

## Important

1. Parts (or all) of this package will almost certainly be renamed and/or merged with the [go-whosonfirst-clone](https://github.com/whosonfirst/go-whosonfirst-clone). The details are still being worked out.
2. There is only local -> remote (S3) synchronization at the moment. See above.
3. There is no retry (for failed uploads) code yet.

## Usage

_Please write me_

## Tools

### wof-s3-sync

_Please finish writing me..._

```
./bin/wof-s3-sync -rate-limit 100000 -bucket data.whosonfirst.org -processes 128 -mode repo /usr/local/data/whosonfirst-data
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

* https://github.com/aws/aws-sdk-go
* https://github.com/whosonfirst/go-whosonfirst-clone