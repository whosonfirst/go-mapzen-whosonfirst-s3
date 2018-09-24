package sync

import (
	"github.com/whosonfirst/go-whosonfirst-index"
	"io"
)

type Sync interface {
	SyncFunc() (index.IndexerFunc, error)
	SyncFile(io.Reader, string) error
}
