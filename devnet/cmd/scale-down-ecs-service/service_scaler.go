// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
)

type serviceScaler struct {
	tickerDuration time.Duration
	cluster        string
	ecs            ecsiface.ECSAPI
}

func newServiceScaler(tickerDuration time.Duration, cluster string, ecs ecsiface.ECSAPI) *serviceScaler {
	return &serviceScaler{
		tickerDuration: tickerDuration,
		cluster:        cluster,
		ecs:            ecs,
	}
}

func (ss serviceScaler) findServiceArns(ctx context.Context, serviceRegex string) (serviceArns []*string, err error) {
	r, err := regexp.Compile(serviceRegex)
	if err != nil {
		return
	}

	var lsi = &ecs.ListServicesInput{
		Cluster: &ss.cluster,
	}
	for {
		var lso *ecs.ListServicesOutput
		lso, err = ss.ecs.ListServicesWithContext(ctx, lsi)
		if err != nil {
			return
		}
		for _, arn := range lso.ServiceArns {
			if r.MatchString(*arn) {
				serviceArns = append(serviceArns, arn)
			}
		}
		if lso.NextToken == nil {
			break
		}
		lsi.NextToken = lso.NextToken
	}

	if len(serviceArns) == 0 {
		err = fmt.Errorf("unable to locate any services with query: %s", serviceRegex)
	}
	return
}

func (ss serviceScaler) drainServices(ctx context.Context, serviceArns []*string) (err error) {
	for _, serviceArn := range serviceArns {
		_, err = ss.ecs.UpdateServiceWithContext(ctx, &ecs.UpdateServiceInput{
			Cluster:      &ss.cluster,
			Service:      serviceArn,
			DesiredCount: aws.Int64(0),
		})
		if err != nil {
			return
		}
	}
	return
}

func (ss serviceScaler) waitForRunningCount(ctx context.Context, serviceArns []*string) (err error) {
	ticker := time.NewTicker(ss.tickerDuration)
	defer ticker.Stop()
main:
	for {
		select {
		case <-ticker.C:
			var dso *ecs.DescribeServicesOutput
			dso, err = ss.ecs.DescribeServicesWithContext(ctx, &ecs.DescribeServicesInput{
				Cluster:  &ss.cluster,
				Services: serviceArns,
			})
			if err != nil {
				break main
			}
			scaledDown := make(map[string]bool)
			for _, service := range dso.Services {
				if service.RunningCount != nil && *service.RunningCount == 0 {
					scaledDown[*service.ServiceArn] = true
				}
			}
			if len(scaledDown) == len(serviceArns) {
				break main
			}
		case <-ctx.Done():
			err = fmt.Errorf("aborting waiting: %w", ctx.Err())
			break main
		}
	}
	return
}

func (ss serviceScaler) scaleServices(ctx context.Context, servicesRegex string) (err error) {
	serviceArns, err := ss.findServiceArns(ctx, servicesRegex)
	if err != nil {
		return
	}

	err = ss.drainServices(ctx, serviceArns)
	if err != nil {
		return
	}

	return ss.waitForRunningCount(ctx, serviceArns)
}
