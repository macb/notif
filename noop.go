package notif

type NoopNotifier struct {
}

func (p *NoopNotifier) Trigger(key, url, desc string, ed EventDetails) (*NotifierResponse, error) {
	return &NotifierResponse{}, nil
}

func (p *NoopNotifier) Resolve(key, url, desc string, ed EventDetails) (*NotifierResponse, error) {
	return &NotifierResponse{}, nil
}
