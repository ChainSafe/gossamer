// Copyright 2023 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package internal

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
)

// ECSAPI is the interface for the ECS API.
type ECSAPI interface {
	ListServicesWithContext(aws.Context, *ecs.ListServicesInput, ...request.Option) (
		*ecs.ListServicesOutput, error)
	UpdateServiceWithContext(aws.Context, *ecs.UpdateServiceInput, ...request.Option) (
		*ecs.UpdateServiceOutput, error)
	DescribeServicesWithContext(aws.Context, *ecs.DescribeServicesInput, ...request.Option) (
		*ecs.DescribeServicesOutput, error)
}
