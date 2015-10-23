package models

import (
	"encoding/json"
	"os"
	"time"

	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/aws"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/api/Godeps/_workspace/src/github.com/ddollar/logger"
)

var PauseNotifications = false

type NotifyEvent struct {
	Action    string            `json:"action"`
	Status    string            `json:"status"`
	Data      map[string]string `json:"data"`
	Timestamp time.Time         `json:"timestamp"`
}

// uniform error handling
func NotifyError(action string, err error, data map[string]string) error {
	data["message"] = err.Error()
	return Notify(action, "error", data)
}

func NotifySuccess(action string, data map[string]string) error {
	return Notify(action, "success", data)
}

func Notify(name, status string, data map[string]string) error {
	if PauseNotifications {
		return nil
	}

	log := logger.New("ns=kernel")
	data["rack"] = os.Getenv("RACK")

	event := &NotifyEvent{
		Action:    name,
		Status:    status,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}

	message, err := json.Marshal(event)
	if err != nil {
		return err
	}

	params := &sns.PublishInput{
		Message:   aws.String(string(message)), // Required
		Subject:   aws.String(name),
		TargetArn: aws.String(NotificationTopic),
	}
	resp, err := SNS().Publish(params)

	if err != nil {
		return err
	}

	log.At("Notfiy").Log("message-id=%q", *resp.MessageId)

	return nil
}