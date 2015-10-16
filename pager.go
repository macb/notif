package notif

import (
	"bytes"
	"encoding/json"
	"net/http"
)

const pdAPI = "https://events.pagerduty.com/generic/2010-04-15/create_event.json"

type Pager struct {
	key    string
	client *http.Client
}

func NewPager(key string, client *http.Client) *Pager {
	if client == nil {
		client = http.DefaultClient
	}

	return &Pager{
		key:    key,
		client: client,
	}
}

type trigger struct {
	ServiceKey  string       `json:"service_key"`
	EventType   string       `json:"event_type"`
	Description string       `json:"description"`
	IncidentKey string       `json:"incident_key"`
	Client      string       `json:"client"`
	ClientURL   string       `json:"client_url"`
	Details     EventDetails `json:"details"`
}

type EventDetails struct {
	Hostname    string `json:"hostname"`
	ServiceName string `json:"service_name"`
	CheckOutput string `json:"check_output"`
}

type PagerResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	IncidentKey string `json:"incident_key"`
}

func (p *Pager) Trigger(incidentKey, url, description string, ed EventDetails) (*PagerResponse, error) {
	t := trigger{
		ServiceKey:  p.key,
		EventType:   "trigger",
		Description: description,
		IncidentKey: incidentKey,
		Client:      "notif",
		ClientURL:   url,
		Details:     ed,
	}

	body, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Post(pdAPI, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	pr := new(PagerResponse)
	err = json.NewDecoder(resp.Body).Decode(pr)
	if err != nil {
		return nil, err
	}

	return pr, err
}

func (p *Pager) Resolve(incidentKey, description string) (*PagerResponse, error) {
	t := trigger{
		ServiceKey:  p.key,
		EventType:   "resolve",
		Description: description,
		IncidentKey: incidentKey,
	}

	body, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Post(pdAPI, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	pr := new(PagerResponse)
	err = json.NewDecoder(resp.Body).Decode(pr)
	if err != nil {
		return nil, err
	}

	return pr, err
}
