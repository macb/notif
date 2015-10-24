package notif

import "log"

type PrintNotifier struct {
}

func (p *PrintNotifier) Trigger(key, url, desc string, ed EventDetails) (*NotifierResponse, error) {
	log.Printf("trigger: %s, %s, %s, %+v\n", key, url, desc, ed)
	return &NotifierResponse{}, nil
}

func (p *PrintNotifier) Resolve(incidentKey, description string) (*NotifierResponse, error) {
	log.Printf("resolved: %s, %s\n", incidentKey, description)
	return &NotifierResponse{}, nil
}
