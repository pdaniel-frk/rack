package aws

import (
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/rack/api/structs"
)

// returns individual server memory, total rack memory
func (p *AWSProvider) CapacityGet() (*structs.Capacity, error) {
	capacity := &structs.Capacity{}

	lres, err := p.ecs().ListContainerInstances(&ecs.ListContainerInstancesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})

	if err != nil {
		return nil, err
	}

	ires, err := p.ecs().DescribeContainerInstances(&ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String(os.Getenv("CLUSTER")),
		ContainerInstances: lres.ContainerInstanceArns,
	})

	if err != nil {
		return nil, err
	}

	for _, instance := range ires.ContainerInstances {
		for _, resource := range instance.RegisteredResources {
			if *resource.Name == "MEMORY" {
				capacity.InstanceMemory = *resource.IntegerValue
				capacity.ClusterMemory += *resource.IntegerValue
			}
			if *resource.Name == "CPU" {
				capacity.InstanceCPU = *resource.IntegerValue
				capacity.ClusterCPU += *resource.IntegerValue
			}
		}
	}

	services, err := p.clusterServices()

	if err != nil {
		return nil, err
	}

	for _, service := range services {
		if len(service.LoadBalancers) > 0 && *service.DesiredCount > capacity.ProcessWidth {
			capacity.ProcessWidth = *service.DesiredCount
		}

		res, err := p.ecs().DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
			TaskDefinition: service.TaskDefinition,
		})

		if err != nil {
			return nil, err
		}

		for _, cd := range res.TaskDefinition.ContainerDefinitions {
			capacity.ProcessCount += *service.DesiredCount
			capacity.ProcessMemory += (*service.DesiredCount * *cd.Memory)
			capacity.ProcessCPU += (*service.DesiredCount * *cd.Cpu)
		}
	}

	// return capacity, concurrency, nil
	return capacity, nil
}

type ECSServices []*ecs.Service

func (services ECSServices) LastEvent() ecs.ServiceEvent {
	e := ecs.ServiceEvent{
		CreatedAt: aws.Time(time.Unix(0, 0)),
	}

	for i := 0; i < len(services); i++ {
		s := services[i]

		if len(s.Events) > 0 && s.Events[0].CreatedAt.After(*e.CreatedAt) {
			e = *s.Events[0]
		}
	}

	return e
}

func (p *AWSProvider) clusterServices() (ECSServices, error) {
	services := ECSServices{}

	lsres, err := p.ecs().ListServices(&ecs.ListServicesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})

	if err != nil {
		return services, err
	}

	dsres, err := p.ecs().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(os.Getenv("CLUSTER")),
		Services: lsres.ServiceArns,
	})

	if err != nil {
		return services, err
	}

	for i := 0; i < len(dsres.Services); i++ {
		services = append(services, dsres.Services[i])
	}

	return services, nil
}
