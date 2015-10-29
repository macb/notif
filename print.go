package notif

import "github.com/Sirupsen/logrus"

type PrintNotifier struct {
}

func (p *PrintNotifier) Trigger(key, url, desc string, ed EventDetails) (*NotifierResponse, error) {
	logrus.WithFields(logrus.Fields{
		"key":           key,
		"url":           url,
		"desc":          desc,
		"event_details": ed,
	}).Info("trigger")
	return &NotifierResponse{}, nil
}

func (p *PrintNotifier) Resolve(key, url, desc string, ed EventDetails) (*NotifierResponse, error) {
	logrus.WithFields(logrus.Fields{
		"key":           key,
		"url":           url,
		"desc":          desc,
		"event_details": ed,
	}).Info("resolved")
	return &NotifierResponse{}, nil
}
