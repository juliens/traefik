package loadbalancer

import (
	"net/http"
)

type Empty struct {
	updaters []func(bool)
}

func (e *Empty) RegisterStatusUpdater(fn func(up bool)) error {
	e.updaters = append(e.updaters, fn)
	return nil
}

func (e *Empty) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	http.Error(rw, "no available server", http.StatusServiceUnavailable)
}

func (e *Empty) PropagateDownStatus() {
	for _, updater := range e.updaters {
		updater(false)
	}
}
