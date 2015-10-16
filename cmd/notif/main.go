package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	consulapi "github.com/hashicorp/consul/api"
	"github.com/macb/notif"
)

func main() {
	pagerkey := flag.String("pager", "", "Pager API string")
	flag.Parse()

	drain := make(chan interface{})
	pager := notif.NewPager(*pagerkey, nil)

	types := []string{"checks"}
	for _, t := range types {
		w, err := notif.NewWatcher("127.0.0.1:8500", t, drain)
		if err != nil {
			panic(err)
		}

		go w.Run()
	}

	cc, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		panic(err)
	}

	for data := range drain {
		switch d := data.(type) {
		case *consulapi.HealthCheck:
			processHealthCheck(pager, cc, d)
		case map[string][]string:
			log.Printf("maps: %#v\n", d)
		case *consulapi.UserEvent:
			log.Printf("what: %#v\n", d)
		default:
			log.Panicf("received unknown type: %T %#v\n", d, data)
		}
	}
}

func processHealthCheck(pager *notif.Pager, cc *consulapi.Client, hc *consulapi.HealthCheck) {
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
		switch hc.Status {
		case "critical":
			notify(pager, hc)
		case "passing":
			resolve(pager, hc)
		}
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

func serviceStatus(node, checkID string) string {
	return fmt.Sprintf("notif/%s/%s", node, checkID)
}

type check struct {
	Status    string    `json:"state"`
	UpdatedAt time.Time `json:"updated_at"`
}

func resolve(pager *notif.Pager, hc *consulapi.HealthCheck) {
	pr, err := pager.Resolve(serviceStatus(hc.Node, hc.CheckID), hc.Output)
	if err != nil {
		panic(err)
	}

	fmt.Printf("(%s) %s: %s\n", pr.IncidentKey, pr.Status, pr.Message)
	fmt.Printf("%s: %s status changed to %s\n", hc.Node, hc.ServiceID, hc.Status)
	fmt.Printf("More info: %#v\n", hc)
}

func notify(pager *notif.Pager, hc *consulapi.HealthCheck) {
	u := fmt.Sprintf("http://127.0.0.1:8500/ui/#/dc1/nodes/%s", hc.Node)
	desc := fmt.Sprintf("%s: %s failing for %s", hc.Node, hc.CheckID, hc.ServiceName)
	pr, err := pager.Trigger(serviceStatus(hc.Node, hc.CheckID), u, desc, notif.EventDetails{
		Hostname:    hc.Node,
		ServiceName: hc.ServiceName,
		CheckOutput: hc.Output,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("(%s) %s: %s\n", pr.IncidentKey, pr.Status, pr.Message)
	fmt.Printf("%s: %s status changed to %s\n", hc.Node, hc.ServiceID, hc.Status)
	fmt.Printf("More info: %#v\n", hc)
}
