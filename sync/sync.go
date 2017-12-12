package sync

import (
	"github.com/whosonfirst/go-whosonfirst-index"
)

type Sync interface {
	SyncFunc() (index.IndexerFunc, error)
	SyncFile(string) error
}
