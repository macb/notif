package notif

type Notifier interface {
	Trigger(incidentKey string, url string, desc string, ed EventDetails) (*NotifierResponse, error)
	Resolve(incidentKey string, url string, desc string, ed EventDetails) (*NotifierResponse, error)
}

type EventDetails struct {
	Hostname    string `json:"hostname"`
	ServiceName string `json:"service_name"`
	CheckName   string `json:"check_name"`
	CheckID     string `json:"check_id"`
	CheckOutput string `json:"check_output"`
}

type NotifierResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	IncidentKey string `json:"incident_key"`
}
