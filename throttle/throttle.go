package throttle

// I will probably move this code in to its own package/repo
// but not today (20171212/thisisaaronland)

type Throttle interface {
	RateLimit() error
}
