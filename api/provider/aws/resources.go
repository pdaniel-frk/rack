package aws

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/convox/rack/api/cache"
	"github.com/convox/rack/api/helpers"
	"github.com/convox/rack/api/structs"
)

func (p *AWSProvider) ResourcesList(app string) (structs.Resources, error) {
	if resources, ok := cache.Get("ListResources", app).(structs.Resources); ok {
		return resources, nil
	}

	stackName := helpers.ShortNameToStackName(app)

	res, err := p.cloudformation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
		StackName: aws.String(stackName),
	})

	if app != stackName && awsError(err) == "ValidationError" {
		res, err = p.cloudformation().DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
			StackName: aws.String(app),
		})
	}

	if err != nil {
		return nil, err
	}

	resources := make(structs.Resources, len(res.StackResources))

	for _, r := range res.StackResources {
		resources[*r.LogicalResourceId] = structs.Resource{
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
