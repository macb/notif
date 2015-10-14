package main

import (
	"encoding/json"
	"fmt"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/macb/notif"
)

func main() {
	drain := make(chan *consulapi.HealthCheck)
	w, err := notif.NewWatcher("127.0.0.1:8500", drain)
	if err != nil {
		panic(err)
	}

	go w.Run()

	cc, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		panic(err)
	}

	for hc := range drain {
		p, _, err := cc.KV().Get(serviceStatus(hc.Node, hc.CheckID), nil)
		if err != nil {
			panic(err)
		}

		s := new(check)
		if p != nil {
			err = json.Unmarshal(p.Value, s)
			if err != nil {
				panic(err)
			}
		} else {
			p = &consulapi.KVPair{
				Key: serviceStatus(hc.Node, hc.CheckID),
			}
		}

		if s.Status != hc.Status {
			notify(hc)
			s.Status = hc.Status
			s.UpdatedAt = time.Now().UTC()
			p.Value, err = json.Marshal(s)
			if err != nil {
				panic(err)
			}

			_, err = cc.KV().Put(p, nil)
			if err != nil {
				panic(err)
			}
		}
	}
}

func serviceStatus(node, checkID string) string {
	return fmt.Sprintf("notif/%s/%s", node, checkID)
}

type check struct {
	Status    string    `json:"state"`
	UpdatedAt time.Time `json:"updated_at"`
}

func notify(hc *consulapi.HealthCheck) {
	fmt.Printf("%s: %s status changed to %s\n", hc.Node, hc.ServiceID, hc.Status)
}
