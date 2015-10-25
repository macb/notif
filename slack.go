package notif

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type SlackNotifier struct {
	username  string
	webhook   string
	iconEmoji string
	channel   string
	client    *http.Client
}

func NewSlackNotifier(username, webhook, iconEmoji, channel string) *SlackNotifier {
	return &SlackNotifier{
		username:  username,
		webhook:   webhook,
		iconEmoji: iconEmoji,
		channel:   channel,
		client:    http.DefaultClient,
	}
}

type slackMessage struct {
	Username    string            `json:"username"`
	Channel     string            `json:"channel"`
	IconEmoji   string            `json:"icon_emoji"`
	Attachments []slackAttachment `json:"attachments"`
}

type slackAttachment struct {
	Fallback string        `json:"fallback"`
	Pretext  string        `json:"pretext"`
	Fields   []slackFields `json:"fields"`
}

type slackFields struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func (s *SlackNotifier) Trigger(incidentKey, url, description string, ed EventDetails) (*NotifierResponse, error) {
	msg := slackMessage{
		Username:  s.username,
		Channel:   s.channel,
		IconEmoji: s.iconEmoji,
		Attachments: []slackAttachment{
			{
				Fallback: fmt.Sprintf("Failing check for %s on %s. See %s", ed.ServiceName, ed.Hostname, url),
				Pretext:  fmt.Sprintf("Failing check for %s on %s", ed.ServiceName, ed.Hostname),
				Fields: []slackFields{
					{
						Title: "URL",
						Value: fmt.Sprintf("<%s>", url),
						Short: false,
					},
				},
			},
		},
	}

	if err := s.sendWebhook(&msg); err != nil {
		return nil, err
	}

	return &NotifierResponse{
		IncidentKey: incidentKey,
		Status:      "success",
		Message:     "slack message posted",
	}, nil
}

func (s *SlackNotifier) Resolve(incidentKey, description string) (*NotifierResponse, error) {
	msg := slackMessage{
		Username:  s.username,
		Channel:   s.channel,
		IconEmoji: s.iconEmoji,
		Attachments: []slackAttachment{
			{
				Fallback: fmt.Sprintf("Passing check for %s.", incidentKey),
				Pretext:  fmt.Sprintf("Passing check for %s. %s", incidentKey, description),
				Fields: []slackFields{
					{
						Title: "Description",
						Value: description,
						Short: false,
					},
				},
			},
		},
	}

	if err := s.sendWebhook(&msg); err != nil {
		return nil, err
	}

	return &NotifierResponse{
		IncidentKey: incidentKey,
		Status:      "success",
		Message:     "slack message posted",
	}, nil
}

func (s *SlackNotifier) sendWebhook(msg *slackMessage) error {
	b := bytes.NewBuffer(nil)
	if err := json.NewEncoder(b).Encode(msg); err != nil {
		return err
	}

	resp, err := s.client.Post(s.webhook, "application/json", b)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	return nil
}
