package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/convox/rack/client"
	"github.com/ddollar/logger"
)

var (
	PauseNotifications = false
	NotificationTopic  = os.Getenv("NOTIFICATION_TOPIC")
	NotificationHost   = os.Getenv("NOTIFICATION_HOST")
)

// uniform error handling
func (p *AWSProvider) NotifyError(action string, err error, data map[string]string) error {
	data["message"] = err.Error()
	return p.Notify(action, "error", data)
}

func (p *AWSProvider) NotifySuccess(action string, data map[string]string) error {
	return p.Notify(action, "success", data)
}

func (p *AWSProvider) Notify(action, status string, data map[string]string) error {
	if PauseNotifications {
		return nil
	}

	log := logger.New("ns=kernel")
	data["rack"] = os.Getenv("RACK")

	event := &client.NotifyEvent{
		Action:    action,
		Status:    status,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}

	message, err := json.Marshal(event)
	if err != nil {
		return err
	}

	fmt.Printf("models EventSend msg=%q\n", message)

	params := &sns.PublishInput{
		Message:   aws.String(string(message)), // Required
		Subject:   aws.String(action),
		TargetArn: aws.String(NotificationTopic),
	}
	resp, err := p.sns().Publish(params)

	if err != nil {
		return err
	}

	log.At("Notify").Log("message-id=%q", *resp.MessageId)

	return nil
}
