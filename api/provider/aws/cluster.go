package aws

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/convox/logger"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) ClusterServices() (*structs.Services, error) {
	var log = logger.New("ns=ClusterServices")

	services := *structs.Services{}

	lsres, err := ECS().ListServices(&ecs.ListServicesInput{
		Cluster: aws.String(os.Getenv("CLUSTER")),
	})
	if err != nil {
		log.Log("at=ListServices err=%q", err)
		return services, err
	}

	dsres, err := ECS().DescribeServices(&ecs.DescribeServicesInput{
		Cluster:  aws.String(os.Getenv("CLUSTER")),
		Services: lsres.ServiceArns,
	})
	if err != nil {
		log.Log("at=ListServices err=%q", err)
		return services, err
	}

	for _, s := range dsres.Services {
		services = append(services, structs.Service{})
	}

	for i := 0; i < len(dsres.Services); i++ {
		services = append(services, dsres.Services[i])
	}

	return services, nil
}
