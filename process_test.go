package notif

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
	ctu "github.com/hashicorp/consul/testutil"
)

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}

type testNotifier struct {
	notified bool
	resolved bool
}

func (t *testNotifier) Trigger(string, string, string, EventDetails) (*NotifierResponse, error) {
	t.notified = true
	return &NotifierResponse{}, nil
}

func (t *testNotifier) Resolve(string, string) (*NotifierResponse, error) {
	t.resolved = true
	return &NotifierResponse{}, nil
}

func (t *testNotifier) reset() {
	t.notified = false
	t.resolved = false
}

func TestProcessHealthCheck(t *testing.T) {
	srv := ctu.NewTestServer(t)
	defer srv.Stop()
	config := *consulapi.DefaultConfig()
	config.Address = srv.HTTPAddr
	cc, err := consulapi.NewClient(&config)
	if err != nil {
		t.Fatal(err)
	}

	drain := make(chan *consulapi.HealthCheck)
	notifier := &testNotifier{}

	pro := NewProcessor(drain, notifier, cc)

	hc := &consulapi.HealthCheck{
		CheckID: "check",
		Node:    "node1",
		Status:  "critical",
	}

	pro.processHealthCheck(hc)

	if !notifier.notified {
		t.Fatal("failed to notify")
	}

	if notifier.resolved {
		t.Fatal("resolved when unexpected")
	}

	if len(pro.hcs) != 1 {
		t.Fatal("failed to memoize check")
	}

	notifier.reset()

	pro.processHealthCheck(hc)
	if notifier.notified {
		t.Fatal("should not have notified")
	}

	if notifier.resolved {
		t.Fatal("resolved when unexpected")
	}
}

func TestProcessHealthCheckWithKnownCheck(t *testing.T) {
	consulKey := "notif/node1/check"

	srv := ctu.NewTestServer(t)
	defer srv.Stop()
	config := *consulapi.DefaultConfig()
	config.Address = srv.HTTPAddr
	cc, err := consulapi.NewClient(&config)
	if err != nil {
		t.Fatal(err)
	}

	drain := make(chan *consulapi.HealthCheck)
	notifier := &testNotifier{}
	pro := NewProcessor(drain, notifier, cc)

	b, err := json.Marshal(check{Status: "critical", UpdatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal("failed to marshal check")
	}
	srv.SetKV(consulKey, b)

	hc := &consulapi.HealthCheck{
		CheckID: "check",
		Node:    "node1",
		Status:  "critical",
	}

	pro.processHealthCheck(hc)
	if notifier.notified {
		t.Fatal("should not have notified")
	}

	if notifier.resolved {
		t.Fatal("resolved when unexpected")
	}

	if len(pro.hcs) != 1 {
		t.Fatal("failed to memoize check")
	}

	notifier.reset()

	pro.processHealthCheck(hc)
	if notifier.notified {
		t.Fatal("should not have notified")
	}

	if notifier.resolved {
		t.Fatal("resolved when unexpected")
	}

	if len(pro.hcs) != 1 {
		t.Fatal("failed to memoize check")
	}
}
