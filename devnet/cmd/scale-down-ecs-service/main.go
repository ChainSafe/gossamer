package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/jessevdk/go-flags"
)

type options struct {
	ServicesRegex   string        `short:"s" long:"services" description:"regex query used to match against AWS service names" required:"true"`          //nolint:lll
	Cluster         string        `short:"c" long:"cluster" description:"ECS cluster name, must be exact match" required:"true"`                         //nolint:lll
	RequestInterval time.Duration `short:"i" long:"interval" description:"Interval between AWS requests when waiting for service to scale" default:"5s"` //nolint:lll
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var opts options
	_, err := flags.Parse(&opts)
	if err != nil {
		log.Panic(err)
	}

	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error)
	go func() {
		ss := newServiceScaler(opts.RequestInterval, opts.Cluster, ecs.New(sess))
		err := ss.scaleServices(ctx, opts.ServicesRegex)
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
