package notif

import (
	"log"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
)

type Watcher struct {
	addr string
	wp   *watch.WatchPlan
}

func NewWatcher(addr string, drain chan<- *consulapi.HealthCheck) (*Watcher, error) {
	wp, err := watch.Parse(map[string]interface{}{
		"type": "checks",
	})
	if err != nil {
		return nil, err
	}

	wp.Handler = func(idx uint64, data interface{}) {
		hcs, ok := data.([]*consulapi.HealthCheck)
		if !ok {
			log.Panicf("received unknown type: %T\n", data)
		}

		for _, hc := range hcs {
			drain <- hc
		}
	}

	return &Watcher{
		addr: addr,
		wp:   wp,
	}, nil
}

func (w *Watcher) Run() {
	w.wp.Run(w.addr)
}
