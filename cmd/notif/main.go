package main

import (
	"github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/ianschenck/envflag"
	"github.com/macb/notif"
)

func main() {
	pagerkey := envflag.String("PAGERDUTY_KEY", "", "Pager API string")
	consulKV := envflag.String("CONSUL_KV_ADDR", "127.0.0.1:8500", "Address to consul to use for KV.")
	consulWatch := envflag.String("CONSUL_WATCH_ADDR", "127.0.0.1:8500", "Address to consul to use for watch.")
	envflag.Parse()

	logrus.SetLevel(logrus.DebugLevel)

	drain := make(chan *consulapi.HealthCheck)
	var pager notif.Notifier
	pager = &notif.PrintNotifier{}
	if *pagerkey != "" {
		pager = notif.NewPager(*pagerkey, nil)
	}

	types := []string{"checks"}
	for _, t := range types {
		w, err := notif.NewWatcher(*consulWatch, t, drain)
		if err != nil {
			panic(err)
		}

		go w.Run()
	}

	config := *consulapi.DefaultConfig()
	config.Address = *consulKV
	cc, err := consulapi.NewClient(&config)
	if err != nil {
		panic(err)
	}
	p := notif.NewProcessor(drain, pager, cc)
	p.Run()
}
