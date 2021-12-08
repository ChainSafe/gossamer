package main

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
)

func Test_findServiceArns(t *testing.T) {
	type args struct {
		cluster      string
		serviceRegex string
	}
	tests := []struct {
		name            string
		args            args
		wantServiceArns []string
		wantErr         bool
	}{
		{
			args: args{
				cluster:      "gssmr-ecs",
				serviceRegex: "gssmr-ecs-(Charlie|Bob)Service-.+$",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotServiceArns, err := findServiceArns(context.Background(), tt.args.cluster, tt.args.serviceRegex)
			if (err != nil) != tt.wantErr {
				t.Errorf("findServiceArns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotServiceArns, tt.wantServiceArns) {
				t.Errorf("findServiceArns() = %v, want %v", gotServiceArns, tt.wantServiceArns)
			}
		})
	}
}

func Test_updateServices(t *testing.T) {
	type args struct {
		cluster     string
		serviceArns []*string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args: args{
				cluster: "gssmr-ecs",
				serviceArns: []*string{
					aws.String("arn:aws:ecs:us-east-2:500822580415:service/gssmr-ecs/gssmr-ecs-CharlieService-J30Gr963xo2y"),
					aws.String("arn:aws:ecs:us-east-2:500822580415:service/gssmr-ecs/gssmr-ecs-BobService-tAQb9CsOMLx7"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := updateServices(context.Background(), tt.args.cluster, tt.args.serviceArns); (err != nil) != tt.wantErr {
				t.Errorf("updateServices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_waitForRunningCount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	type args struct {
		ctx         context.Context
		cluster     string
		serviceArns []*string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		sidecar func(t *testing.T)
	}{
		{
			args: args{
				cluster: "gssmr-ecs",
				serviceArns: []*string{
					aws.String("arn:aws:ecs:us-east-2:500822580415:service/gssmr-ecs/gssmr-ecs-CharlieService-J30Gr963xo2y"),
					aws.String("arn:aws:ecs:us-east-2:500822580415:service/gssmr-ecs/gssmr-ecs-BobService-tAQb9CsOMLx7"),
				},
				ctx: ctx,
			},
			sidecar: func(t *testing.T) {
				<-time.After(6 * time.Second)
				cancel()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.sidecar != nil {
				go tt.sidecar(t)
			}
			if err := waitForRunningCount(tt.args.ctx, tt.args.cluster, tt.args.serviceArns); (err != nil) != tt.wantErr {
				t.Errorf("waitUntilScaledDown() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
