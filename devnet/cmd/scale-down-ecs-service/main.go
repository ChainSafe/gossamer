package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/jessevdk/go-flags"
)

var (
	svc *ecs.ECS
)

func init() {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc = ecs.New(sess)
}

func findServiceArns(ctx context.Context, cluster, serviceRegex string) (serviceArns []*string, err error) {
	r, err := regexp.Compile(serviceRegex)
	if err != nil {
		return
	}

	var lsi = &ecs.ListServicesInput{
		Cluster: &cluster,
	}
	for {
		var lso *ecs.ListServicesOutput
		lso, err = svc.ListServicesWithContext(ctx, lsi)
		if err != nil {
			return
		}
		for _, arn := range lso.ServiceArns {
			if r.MatchString(*arn) {
				serviceArns = append(serviceArns, arn)
			}
		}
		if lso.NextToken != nil {
			lsi.NextToken = lso.NextToken
		} else {
			break
		}
	}
	return
}

func updateServices(ctx context.Context, cluster string, serviceArns []*string) (err error) {
	for _, serviceArn := range serviceArns {
		_, err = svc.UpdateServiceWithContext(ctx, &ecs.UpdateServiceInput{
			Cluster:      &cluster,
			Service:      serviceArn,
			DesiredCount: aws.Int64(0),
		})
		if err != nil {
			return
		}
	}
	return
}

func waitForRunningCount(ctx context.Context, cluster string, serviceArns []*string) (err error) {
	ticker := time.NewTicker(time.Second * 5)
main:
	for {
		select {
		case <-ticker.C:
			var dso *ecs.DescribeServicesOutput
			dso, err = svc.DescribeServicesWithContext(ctx, &ecs.DescribeServicesInput{
				Cluster:  &cluster,
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
			err = fmt.Errorf("aborting waiting, received done from context")
			break main
		}
	}
	return
}

func scaleServices(ctx context.Context, cluster string, servicesRegex string) (err error) {
	serviceArns, err := findServiceArns(ctx, cluster, servicesRegex)
	if err != nil {
		return
	}

	err = updateServices(ctx, cluster, serviceArns)
	if err != nil {
		return
	}

	err = waitForRunningCount(ctx, cluster, serviceArns)
	return
}

type options struct {
	ServicesRegex string `short:"s" long:"services" description:"regex query used to match against AWS service names" required:"true"` //nolint:lll
	Cluster       string `short:"c" long:"cluster" description:"ECS cluster name, must be exact match"`
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var opts options
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error)
	go func() {
		err = scaleServices(ctx, opts.Cluster, opts.ServicesRegex)
		done <- err
	}()

	for {
		select {
		case err := <-done:
			if err != nil {
				log.Panic(err)
			}
			// happy path
			return
		case <-sigs:
			cancel()
		}
	}
}
