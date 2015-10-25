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
	bootstrap := envflag.Bool("NOTIF_BOOTSTRAP", false, "Starts the daemon in bootstrap mode. This prevents it from emitting any notifications and should be used to pre-populate the KV store with the current state of the world.")

	slackWebhook := envflag.String("SLACK_WEBHOOK_URL", "", "The webhook URL for slack.")
	slackUsername := envflag.String("SLACK_USERNAME", "notif", "The username for the slack webhook to be posted as.")
	slackChannel := envflag.String("SLACK_CHANNEL", "", "The channel for the slack webhook to be post to.")
	slackIcon := envflag.String("SLACK_ICON", "", "The icon to use when posting the slack webhook.")

	envflag.Parse()

	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.JSONFormatter{})

	drain := make(chan *consulapi.HealthCheck)

	var pager notif.Notifier
	if *bootstrap {
		pager = &notif.NoopNotifier{}
	} else if *pagerkey != "" {
		pager = notif.NewPager(*pagerkey, nil)
	} else if *slackWebhook != "" {
		pager = notif.NewSlackNotifier(*slackUsername, *slackWebhook, *slackIcon, *slackChannel)
	} else {
		pager = &notif.PrintNotifier{}
	}

	w, err := notif.NewWatcher(*consulWatch, "checks", drain)
	if err != nil {
		logrus.WithFields(logrus.Fields{"err": err}).Error("faild to build watcher")
	}

	go w.Run()

	config := *consulapi.DefaultConfig()
	config.Address = *consulKV
	cc, err := consulapi.NewClient(&config)
	if err != nil {
		panic(err)
	}
	p := notif.NewProcessor(drain, pager, cc)
	p.Run()
}
