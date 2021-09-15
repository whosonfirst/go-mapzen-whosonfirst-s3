package sync

import (
	"github.com/whosonfirst/go-whosonfirst-iterate/emitter"
	"io"
)

type Sync interface {
	SyncFunc() (emitter.EmitterCallbackFunc, error)
	SyncFile(io.Reader, string) error
}
