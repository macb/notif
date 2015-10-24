package notif

import (
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/watch"
)

type Watcher struct {
	addr string
	wp   *watch.WatchPlan
}

func NewWatcher(addr string, watchType string, drain chan<- *consulapi.HealthCheck) (*Watcher, error) {
	wp, err := watch.Parse(map[string]interface{}{
		"type": watchType,
	})
	if err != nil {
		return nil, err
	}

	wp.Handler = func(idx uint64, data interface{}) {
		switch d := data.(type) {
		case []*consulapi.HealthCheck:
			for _, i := range d {
				drain <- i
			}
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

type Notifier interface {
	Trigger(incidentKey string, url string, desc string, ed EventDetails) (*NotifierResponse, error)
	Resolve(incidentKey string, output string) (*NotifierResponse, error)
}
