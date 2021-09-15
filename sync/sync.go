package sync

import (
	"context"
	"github.com/whosonfirst/go-whosonfirst-iterate/emitter"
	"io"
)

type Sync interface {
	SyncFunc() (emitter.EmitterCallbackFunc, error)
	SyncFile(context.Context, io.Reader, string) error
}
