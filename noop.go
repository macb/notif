package notif

type NoopNotifier struct {
}

func (p *NoopNotifier) Trigger(key, url, desc string, ed EventDetails) (*NotifierResponse, error) {
	return &NotifierResponse{}, nil
}

func (p *NoopNotifier) Resolve(incidentKey, description string) (*NotifierResponse, error) {
	return &NotifierResponse{}, nil
}
