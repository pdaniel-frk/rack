package main

import (
	"time"

	"github.com/convox/rack/api/provider"
	"github.com/convox/rack/api/workers"
)

func main() {
	go workers.StartAutoscale()
	go workers.StartCluster()
	go provider.MonitorHeartbeat()
	go workers.StartServicesCapacity()

	for {
		time.Sleep(1 * time.Hour)
	}
}
