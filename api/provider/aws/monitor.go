package aws

import (
	"os"
	"time"

	"github.com/convox/rack/api/helpers"
	"github.com/ddollar/logger"
)

func (p *AWSProvider) MonitorHeartbeat() {
	log := logger.New("ns=heartbeat")

	defer helpers.RecoverWith(func(err error) {
		helpers.Error(log, err)
	})

	p.heartbeat(log)

	for _ = range time.Tick(1 * time.Hour) {
		p.heartbeat(log)
	}
}

func (p *AWSProvider) heartbeat(log *logger.Logger) {
	system, err := p.SystemGet()
	if err != nil {
		log.Error(err)
		return
	}

	apps, err := p.AppList()
	if err != nil {
		log.Error(err)
		return
	}

	helpers.TrackEvent("kernel-heartbeat", map[string]interface{}{
		"app_count":      len(apps),
		"instance_count": system.Count,
		"instance_type":  system.Type,
		"region":         os.Getenv("AWS_REGION"),
		"version":        system.Version,
	})
}
