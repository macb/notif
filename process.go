package notif

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
)

type Processor struct {
	drain    <-chan *consulapi.HealthCheck
	notifier Notifier
	// hcs memoizes healthchecks to prevent KV lookups on unchanged health checks.
	// NOTE(macb): May go away with 0.6 assuming the watch functionality only returns
	// the changed healthchecks instead of _all_ healthchecks.
	hcs map[string]*check

	cc *consulapi.Client

	log *logrus.Entry
}

func NewProcessor(drain <-chan *consulapi.HealthCheck, notifier Notifier, cc *consulapi.Client) *Processor {
	return &Processor{
		drain:    drain,
		notifier: notifier,
		hcs:      make(map[string]*check),
		cc:       cc,
		log:      logrus.WithField("system", "processor"),
	}
}

func (p *Processor) Run() {
	for hc := range p.drain {
		p.processHealthCheck(hc)
	}
}

func (p *Processor) processHealthCheck(hc *consulapi.HealthCheck) {
	if ignoredCheckID(hc.CheckID) {
		return
	}

	ss := serviceStatus(hc.Node, hc.CheckID)
	lockKey := serviceStatus(hc.Node, hc.CheckID) + "/lock"

	lock, err := p.cc.LockKey(lockKey)
	if err != nil {
		p.log.WithFields(logrus.Fields{
			"err":        err,
			"consul.key": lockKey,
		}).Error("failed to build lock key")
		return
	}

	_, err = lock.Lock(nil)
	if err != nil {
		p.log.WithFields(logrus.Fields{
			"err":        err,
			"consul.key": lockKey,
		}).Error("failed to get lock")
		return
	}
	defer lock.Unlock()
	p.log.WithFields(logrus.Fields{
		"consul.key": lockKey,
	}).Debug("locked key")

	storedCheck, process, err := p.findCheck(hc)
	if err != nil {
		p.log.WithFields(logrus.Fields{
			"err":            err,
			"service.status": ss,
		}).Error("failed to find check")
		return
	}

	if !process {
		p.log.WithFields(logrus.Fields{
			"process":        process,
			"service.status": ss,
		}).Debug("not processing healthcheck")
		return
	}

	switch hc.Status {
	case "critical":
		p.log.WithFields(logrus.Fields{
			"service.status": ss,
		}).Debug("notifying")
		_, err := p.notify(hc)
		if err != nil {
			// TODO(macb): This is bad. We didn't notify about a failure. Let's not
			// memoize and instead just get out quick. That way next time we'll try to
			// notify.
			p.log.WithFields(logrus.Fields{
				"err":          err,
				"incident.key": ss,
			}).Error("failed to notify")
			return
		}
	case "passing":
		// We don't want to resolve if the stored check does not record having triggered.
		// This is hit mostly when a new service is introduced.
		if storedCheck.Status != "" {
			p.log.WithFields(logrus.Fields{
				"service.status": ss,
			}).Debug("resolving")
			_, err := p.resolve(hc)
			if err != nil {
				p.log.WithFields(logrus.Fields{
					"err":          err,
					"incident.key": ss,
				}).Error("failed to notify")
				return
			}
		}
	}

	storedCheck.Status = hc.Status
	storedCheck.UpdatedAt = time.Now().UTC()

	_ = p.storeCheck(ss, storedCheck)

	return
}

func (p *Processor) notify(hc *consulapi.HealthCheck) (*NotifierResponse, error) {
	// TODO(macb): Supplied from the processor. Hardcoded values bad.
	u := fmt.Sprintf("http://127.0.0.1:8500/ui/#/dc1/nodes/%s", hc.Node)

	// TODO(macb): Allow format to be templatable.
	desc := fmt.Sprintf("%s: %s failing for %s", hc.Node, hc.CheckID, hc.ServiceName)

	ik := serviceStatus(hc.Node, hc.CheckID)
	return p.notifier.Trigger(ik, u, desc, EventDetails{
		Hostname:    hc.Node,
		ServiceName: hc.ServiceName,
		CheckName:   hc.Name,
		CheckID:     hc.CheckID,
		CheckOutput: hc.Output,
	})
}

func (p *Processor) resolve(hc *consulapi.HealthCheck) (*NotifierResponse, error) {
	// TODO(macb): Supplied from the processor. Hardcoded values bad.
	u := fmt.Sprintf("http://127.0.0.1:8500/ui/#/dc1/nodes/%s", hc.Node)

	// TODO(macb): Allow format to be templatable.
	desc := fmt.Sprintf("%s: %s resolved for %s", hc.Node, hc.CheckID, hc.ServiceName)

	ik := serviceStatus(hc.Node, hc.CheckID)
	return p.notifier.Resolve(ik, u, desc, EventDetails{
		Hostname:    hc.Node,
		ServiceName: hc.ServiceName,
		CheckName:   hc.Name,
		CheckID:     hc.CheckID,
		CheckOutput: hc.Output,
	})
}

// Translates the consul health check into a notif check and determines if we should process it.
func (p *Processor) findCheck(hc *consulapi.HealthCheck) (*check, bool, error) {
	ss := serviceStatus(hc.Node, hc.CheckID)

	mc, ok := p.hcs[ss]
	switch {
	case !ok:
		p.log.WithField("memoize.key", ss).Debug("failed to find memoized check")
	case ok && mc.Status == hc.Status:
		return nil, false, nil
	default:
		p.log.WithFields(logrus.Fields{
			"memoize.key": ss,
			"mc.status":   mc.Status,
			"hc.status":   hc.Status,
		}).Debug("memoized check did not match healthcheck")
	}

	// Our cached version does not match the check we received. Lookup from source of
	// truth to ensure nothing else already processed the check.
	p.log.WithField("consul.key", ss).Debug("fetching kv")
	res, _, err := p.cc.KV().Get(ss, nil)
	if err != nil {
		p.log.WithFields(logrus.Fields{
			"err":        err,
			"consul.key": ss,
		}).Error("error fetching kv")
		return nil, false, err
	}

	processCheck := true
	storedCheck := new(check)
	if res != nil {
		err = json.Unmarshal(res.Value, storedCheck)
		if err != nil {
			p.log.WithField("err", err).Error("failed to unmarshal check")
		}
	}

	if storedCheck.Status == hc.Status {
		processCheck = false
		p.log.WithFields(logrus.Fields{
			"memoize.key": ss,
			"sc.status":   storedCheck.Status,
			"hc.status":   hc.Status,
		}).Debug("stored check matched healthcheck")
		p.memoize(ss, storedCheck)
	}

	return storedCheck, processCheck, nil
}

func (p *Processor) storeCheck(key string, ck *check) (err error) {
	p.log.WithFields(logrus.Fields{
		"consul.key": key,
	}).Debug("storing check")

	res := &consulapi.KVPair{
		Key: key,
	}

	res.Value, err = json.Marshal(ck)
	if err != nil {
		p.log.WithFields(logrus.Fields{
			"err":        err,
			"consul.key": key,
		}).Error("failed to marshal check")

		return err
	}

	_, err = p.cc.KV().Put(res, nil)
	if err != nil {
		p.log.WithFields(logrus.Fields{
			"err":        err,
			"consul.key": key,
		}).Error("failed to store check")

		return err
	}

	p.memoize(key, ck)

	return nil
}

func (p *Processor) memoize(key string, ck *check) {
	p.log.WithFields(logrus.Fields{
		"memoize.key":      key,
		"check.status":     ck.Status,
		"check.updated_at": ck.UpdatedAt,
	}).Debug("memoizing check")
	p.hcs[key] = ck
}

type check struct {
	Status    string    `json:"state"`
	UpdatedAt time.Time `json:"updated_at"`
}
