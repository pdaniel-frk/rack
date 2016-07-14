package aws

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/helpers"
	"github.com/ddollar/logger"
)

var lastASGActivity = time.Now()

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

func (p *AWSProvider) MonitorCluster() {
	var log = logger.New("ns=cluster_monitor")

	defer helpers.RecoverWith(func(err error) {
		helpers.Error(log, err)
	})

	for _ = range time.Tick(1 * time.Minute) {
		log.Log("tick")

		instances := ec2Instances{}

		err := describeASG(instances, p)
		if err != nil {
			log.Error(err)
			continue
		}

		err = describeECS(instances, p)
		if err != nil {
			log.Error(err)
			continue
		}

		// Test if ASG Instance is registered and connected in ECS cluster
		for k, i := range instances {
			if !i.ASG {
				// TODO: Rogue instance?! Terminate?
				continue
			}

			if !i.ECS {
				// Not registered or not connected => set Unhealthy
				_, err := p.autoscaling().SetInstanceHealth(
					&autoscaling.SetInstanceHealthInput{
						HealthStatus:             aws.String("Unhealthy"),
						InstanceId:               aws.String(i.Id),
						ShouldRespectGracePeriod: aws.Bool(true),
					},
				)

				i.Unhealthy = true
				instances[k] = i

				if err != nil {
					log.Error(err)
					continue
				}

				// log for humans
				fmt.Printf("who=\"convox/monitor\" what=\"marked instance %s unhealthy\" why=\"ECS reported agent disconnected\"\n", i.Id)
			}
		}

		log.Log(instances.log())
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

type ec2Instance struct {
	Id string

	ASG    bool
	Check  bool
	Docker bool
	ECS    bool

	Unhealthy bool
}

type ec2Instances map[string]ec2Instance

func describeASG(instances ec2Instances, p *AWSProvider) error {
	resources, err := p.ResourcesList(os.Getenv("RACK"))
	if err != nil {
		return err
	}

	res, err := p.autoscaling().DescribeAutoScalingGroups(
		&autoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{
				aws.String(resources["Instances"].Id),
			},
		},
	)
	if err != nil {
		return err
	}

	for _, i := range res.AutoScalingGroups[0].Instances {
		instance := instances[*i.InstanceId]

		instance.Id = *i.InstanceId
		instance.ASG = *i.LifecycleState == "InService"

		instances[*i.InstanceId] = instance
	}

	// describe and log every recent ASG activity
	dres, err := p.autoscaling().DescribeScalingActivities(
		&autoscaling.DescribeScalingActivitiesInput{
			AutoScalingGroupName: aws.String(resources["Instances"].Id),
		},
	)
	if err != nil {
		return nil
	}

	for _, a := range dres.Activities {
		if lastASGActivity.Before(*a.StartTime) {
			fmt.Printf("who=\"EC2/ASG\" what=%q why=%q\n", *a.Description, *a.Cause)
			lastASGActivity = *a.StartTime
		}
	}

	return nil
}

func describeECS(instances ec2Instances, p *AWSProvider) error {
	res, err := p.ecs().ListContainerInstances(
		&ecs.ListContainerInstancesInput{
			Cluster: aws.String(os.Getenv("CLUSTER")),
		},
	)

	if err != nil {
		return err
	}

	dres, err := p.ecs().DescribeContainerInstances(
		&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(os.Getenv("CLUSTER")),
			ContainerInstances: res.ContainerInstanceArns,
		},
	)

	if err != nil {
		return err
	}

	for _, i := range dres.ContainerInstances {
		in := instances[*i.Ec2InstanceId]

		in.Id = *i.Ec2InstanceId
		in.ECS = *i.AgentConnected

		instances[*i.Ec2InstanceId] = in
	}

	return nil
}

func (instances ec2Instances) log() string {
	var asgIds, ecsIds, unhealthyIds []string

	for _, i := range instances {
		if i.ASG {
			asgIds = append(asgIds, i.Id)
		}

		if i.ECS {
			ecsIds = append(ecsIds, i.Id)
		}

		if i.Unhealthy {
			unhealthyIds = append(unhealthyIds, i.Id)
		}
	}

	sort.Strings(asgIds)
	sort.Strings(ecsIds)
	sort.Strings(unhealthyIds)

	return fmt.Sprintf("count=%v connected='%v' healthy='%v' marked='%s'",
		len(instances),
		strings.Join(ecsIds, ","),
		strings.Join(asgIds, ","),
		strings.Join(unhealthyIds, ","),
	)
}
