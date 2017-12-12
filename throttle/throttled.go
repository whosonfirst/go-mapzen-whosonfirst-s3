package throttle

import (
	"errors"
	"github.com/throttled/throttled"
	"github.com/throttled/throttled/store/memstore"
	"time"
)

type ThrottledThrottle struct {
	Throttle
	throttled *throttled.GCRARateLimiter
	tts       int
}

func NewThrottledThrottle(per_min int) (Throttle, error) {

	st, err := memstore.New(65536)

	if err != nil {
		return nil, err
	}

	burst := int(float64(per_min) / 10.)
	quota := throttled.RateQuota{throttled.PerMin(per_min), burst}

	th, err := throttled.NewGCRARateLimiter(st, quota)

	if err != nil {
		return nil, err
	}

	t := ThrottledThrottle{
		throttled: th,
		tts:       100,
	}

	return &t, nil
}

func (t *ThrottledThrottle) RateLimit() error {

	ms := t.tts

	tries := 0
	max_tries := 1000

	for {

		limited, _, err := t.throttled.RateLimit("bucket", 1)

		if err != nil {
			return err
		}

		if limited {
			time.Sleep(time.Duration(ms) * time.Millisecond)
			ms += int(float64(ms) / 10.0)
		} else {
			break
		}

		tries += 1

		if tries >= max_tries {
			return errors.New("Rate limits exceeded max tries")
		}
	}

	return nil
}
