package models

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/api/cache"
	"github.com/convox/rack/api/helpers"
)

type Resource struct {
	Id   string
	Name string

	Reason string
	Status string
	Type   string

	Time time.Time
}

type Resources map[string]Resource

func ListResources(app string) (Resources, error) {
	if resources, ok := cache.Get("ListResources", app).(Resources); ok {
		return resources, nil
	}

	stackName := shortNameToStackName(app)

	res, err := CloudFormation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stackName),
	})

	if app != stackName && awsError(err) == "ValidationError" {
		res, err = CloudFormation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
			StackName: aws.String(app),
		})
	}

	if err != nil {
		return nil, err
	}

	resources := make(Resources, len(res.StackResources))

	for _, r := range res.StackResources {
		resources[*r.LogicalResourceId] = Resource{
			Id:     helpers.CoalesceS(r.PhysicalResourceId, ""),
			Name:   helpers.CoalesceS(r.LogicalResourceId, ""),
			Reason: helpers.CoalesceS(r.ResourceStatusReason, ""),
			Status: helpers.CoalesceS(r.ResourceStatus, ""),
			Type:   helpers.CoalesceS(r.ResourceType, ""),
			Time:   helpers.CoalesceT(r.Timestamp),
		}
	}

	err = cache.Set("ListResources", app, resources, 15*time.Second)

	if err != nil {
		return nil, err
	}

	return resources, nil
}
