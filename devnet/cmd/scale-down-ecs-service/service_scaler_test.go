// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/devnet/cmd/scale-down-ecs-service/mocks"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/ecs/ecsiface"
	"github.com/stretchr/testify/mock"
)

//go:generate mockery --srcpkg=github.com/aws/aws-sdk-go/service/ecs/ecsiface --name ECSAPI --case underscore

func Test_serviceScaler_findServiceArns(t *testing.T) {
	mockECS := mocks.ECSAPI{}
	mockECS.
		On("ListServicesWithContext", mock.Anything, &ecs.ListServicesInput{
			Cluster: aws.String("someCluster"),
		}).Return(&ecs.ListServicesOutput{
		ServiceArns: []*string{
			aws.String("someArn0"),
			aws.String("someArn1"),
		},
		NextToken: aws.String("someNextToken")}, nil).Once().
		On("ListServicesWithContext", mock.Anything, &ecs.ListServicesInput{
			Cluster:   aws.String("someCluster"),
			NextToken: aws.String("someNextToken"),
		}).Return(&ecs.ListServicesOutput{
		ServiceArns: []*string{
			aws.String("someArn2"),
			aws.String("someArn3"),
		}}, nil).Once().
		On("ListServicesWithContext", mock.Anything, &ecs.ListServicesInput{
			Cluster: aws.String("someErrCluster"),
		}).Return(nil, fmt.Errorf("someErr")).Once().
		On("ListServicesWithContext", mock.Anything, &ecs.ListServicesInput{
			Cluster: aws.String("someEmptyCluster"),
		}).Return(&ecs.ListServicesOutput{}, nil).Once()

	type fields struct {
		tickerDuration time.Duration
		cluster        string
		ecs            ecsiface.ECSAPI
	}

	type args struct {
		ctx          context.Context
		serviceRegex string
	}
	tests := []struct {
		name            string
		fields          fields
		args            args
		wantServiceArns []*string
		wantErr         bool
	}{
		{
			name: "with next token",
			fields: fields{
				cluster: "someCluster",
				ecs:     &mockECS,
			},
			args: args{
				ctx:          context.Background(),
				serviceRegex: "someArn",
			},
			wantServiceArns: []*string{
				aws.String("someArn0"),
				aws.String("someArn1"),
				aws.String("someArn2"),
				aws.String("someArn3"),
			},
		},
		{
			name: "ListServicesWithContext err",
			fields: fields{
				cluster: "someErrCluster",
				ecs:     &mockECS,
			},
			args: args{
				ctx:          context.Background(),
				serviceRegex: "someArn",
			},
			wantErr: true,
		},
		{
			name: "no services err",
			fields: fields{
				cluster: "someEmptyCluster",
				ecs:     &mockECS,
			},
			args: args{
				ctx:          context.Background(),
				serviceRegex: "someArn",
			},
			wantErr: true,
		},
		{
			name: "regex err",
			fields: fields{
				ecs: &mockECS,
			},
			args: args{
				ctx:          context.Background(),
				serviceRegex: "BOOM\\",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := serviceScaler{
				tickerDuration: tt.fields.tickerDuration,
				cluster:        tt.fields.cluster,
				ecs:            tt.fields.ecs,
			}
			gotServiceArns, err := ss.findServiceArns(tt.args.ctx, tt.args.serviceRegex)
			if (err != nil) != tt.wantErr {
				t.Errorf("serviceScaler.findServiceArns() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotServiceArns, tt.wantServiceArns) {
				t.Errorf("serviceScaler.findServiceArns() = %v, want %v", gotServiceArns, tt.wantServiceArns)
			}
		})
	}
}

func Test_serviceScaler_updateServices(t *testing.T) {
	mockECS := mocks.ECSAPI{}
	mockECS.
		On("UpdateServiceWithContext", mock.Anything, &ecs.UpdateServiceInput{
			Cluster:      aws.String("someCluster"),
			Service:      aws.String("someArn0"),
			DesiredCount: aws.Int64(0),
		}).Return(&ecs.UpdateServiceOutput{}, nil).Once().
		On("UpdateServiceWithContext", mock.Anything, &ecs.UpdateServiceInput{
			Cluster:      aws.String("someCluster"),
			Service:      aws.String("someArn1"),
			DesiredCount: aws.Int64(0),
		}).Return(&ecs.UpdateServiceOutput{}, nil).Once().
		On("UpdateServiceWithContext", mock.Anything, &ecs.UpdateServiceInput{
			Cluster:      aws.String("someErrCluster"),
			Service:      aws.String("someArn0"),
			DesiredCount: aws.Int64(0),
		}).Return(nil, fmt.Errorf("some Error")).Once()

	type fields struct {
		tickerDuration time.Duration
		cluster        string
		ecs            ecsiface.ECSAPI
	}
	type args struct {
		ctx         context.Context
		serviceArns []*string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				cluster: "someCluster",
				ecs:     &mockECS,
			},
			args: args{
				ctx: context.Background(),
				serviceArns: []*string{
					aws.String("someArn0"),
					aws.String("someArn1"),
				},
			},
		},
		{
			name: "UpdateServiceWithContext err",
			fields: fields{
				cluster: "someErrCluster",
				ecs:     &mockECS,
			},
			args: args{
				ctx: context.Background(),
				serviceArns: []*string{
					aws.String("someArn0"),
					aws.String("someArn1"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := serviceScaler{
				tickerDuration: tt.fields.tickerDuration,
				cluster:        tt.fields.cluster,
				ecs:            tt.fields.ecs,
			}
			if err := ss.updateServices(tt.args.ctx, tt.args.serviceArns); (err != nil) != tt.wantErr {
				t.Errorf("serviceScaler.updateServices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_serviceScaler_waitForRunningCount(t *testing.T) {
	mockECS := mocks.ECSAPI{}
	mockECS.
		On("DescribeServicesWithContext", mock.Anything, &ecs.DescribeServicesInput{
			Cluster: aws.String("someCluster"),
			Services: []*string{
				aws.String("someArn0"),
				aws.String("someArn1"),
			},
		}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				RunningCount: aws.Int64(0),
				ServiceArn:   aws.String("someArn0"),
			},
			{
				RunningCount: aws.Int64(0),
				ServiceArn:   aws.String("someArn1"),
			},
		}}, nil).Once().
		On("DescribeServicesWithContext", mock.Anything, &ecs.DescribeServicesInput{
			Cluster: aws.String("someErrorCluster"),
			Services: []*string{
				aws.String("someArn0"),
				aws.String("someArn1"),
			},
		}).Return(nil, fmt.Errorf("someError")).Once()

	ctx, cancel := context.WithCancel(context.Background())
	mockECSCancel := mocks.ECSAPI{}
	mockECSCancel.
		On("DescribeServicesWithContext", mock.Anything, &ecs.DescribeServicesInput{
			Cluster: aws.String("someCluster"),
			Services: []*string{
				aws.String("someArn0"),
				aws.String("someArn1"),
			},
		}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				RunningCount: aws.Int64(1),
				ServiceArn:   aws.String("someArn0"),
			},
			{
				RunningCount: aws.Int64(1),
				ServiceArn:   aws.String("someArn1"),
			},
		}}, nil).Run(
		func(args mock.Arguments) {
			go func() {
				// should trigger before 10ms ticker
				<-time.After(2 * time.Millisecond)
				cancel()
			}()
		}).Once()

	type fields struct {
		tickerDuration time.Duration
		cluster        string
		ecs            ecsiface.ECSAPI
	}
	type args struct {
		ctx         context.Context
		serviceArns []*string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "already 0 zero running count",
			fields: fields{
				tickerDuration: time.Nanosecond,
				cluster:        "someCluster",
				ecs:            &mockECS,
			},
			args: args{
				ctx: context.Background(),
				serviceArns: []*string{
					aws.String("someArn0"),
					aws.String("someArn1"),
				},
			},
		},
		{
			name: "DescribeServicesWithContext error",
			fields: fields{
				tickerDuration: time.Nanosecond,
				cluster:        "someErrorCluster",
				ecs:            &mockECS,
			},
			args: args{
				ctx: context.Background(),
				serviceArns: []*string{
					aws.String("someArn0"),
					aws.String("someArn1"),
				},
			},
			wantErr: true,
		},
		{
			name: "context cancel err",
			fields: fields{
				tickerDuration: 10 * time.Millisecond,
				cluster:        "someCluster",
				ecs:            &mockECSCancel,
			},
			args: args{
				ctx: ctx,
				serviceArns: []*string{
					aws.String("someArn0"),
					aws.String("someArn1"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := serviceScaler{
				tickerDuration: tt.fields.tickerDuration,
				cluster:        tt.fields.cluster,
				ecs:            tt.fields.ecs,
			}
			if err := ss.waitForRunningCount(tt.args.ctx, tt.args.serviceArns); (err != nil) != tt.wantErr {
				t.Errorf("serviceScaler.waitForRunningCount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_newServiceScaler(t *testing.T) {
	type args struct {
		tickerDuration time.Duration
		cluster        string
		ecs            ecsiface.ECSAPI
	}
	tests := []struct {
		name string
		args args
		want *serviceScaler
	}{
		{
			name: "already 0 zero running count",
			want: &serviceScaler{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newServiceScaler(tt.args.tickerDuration, tt.args.cluster, tt.args.ecs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newServiceScaler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_serviceScaler_scaleServices(t *testing.T) {
	mockECS := mocks.ECSAPI{}
	mockECS.
		On("ListServicesWithContext", mock.Anything, &ecs.ListServicesInput{
			Cluster: aws.String("someCluster"),
		}).Return(&ecs.ListServicesOutput{
		ServiceArns: []*string{
			aws.String("someArn0"),
			aws.String("someArn1"),
		}}, nil).Once().
		On("UpdateServiceWithContext", mock.Anything, &ecs.UpdateServiceInput{
			Cluster:      aws.String("someCluster"),
			Service:      aws.String("someArn0"),
			DesiredCount: aws.Int64(0),
		}).Return(&ecs.UpdateServiceOutput{}, nil).Once().
		On("UpdateServiceWithContext", mock.Anything, &ecs.UpdateServiceInput{
			Cluster:      aws.String("someCluster"),
			Service:      aws.String("someArn1"),
			DesiredCount: aws.Int64(0),
		}).Return(&ecs.UpdateServiceOutput{}, nil).Once().
		On("DescribeServicesWithContext", mock.Anything, &ecs.DescribeServicesInput{
			Cluster: aws.String("someCluster"),
			Services: []*string{
				aws.String("someArn0"),
				aws.String("someArn1"),
			},
		}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{
				RunningCount: aws.Int64(0),
				ServiceArn:   aws.String("someArn0"),
			},
			{
				RunningCount: aws.Int64(0),
				ServiceArn:   aws.String("someArn1"),
			},
		}}, nil).Once()

	findServiceArnsErrECS := mocks.ECSAPI{}
	findServiceArnsErrECS.
		On("ListServicesWithContext", mock.Anything, &ecs.ListServicesInput{
			Cluster: aws.String("someCluster"),
		}).Return(nil, fmt.Errorf("someError")).Once()

	updateServicesErrECS := mocks.ECSAPI{}
	updateServicesErrECS.
		On("ListServicesWithContext", mock.Anything, &ecs.ListServicesInput{
			Cluster: aws.String("someCluster"),
		}).Return(&ecs.ListServicesOutput{
		ServiceArns: []*string{
			aws.String("someArn0"),
			aws.String("someArn1"),
		}}, nil).Once().
		On("UpdateServiceWithContext", mock.Anything, &ecs.UpdateServiceInput{
			Cluster:      aws.String("someCluster"),
			Service:      aws.String("someArn0"),
			DesiredCount: aws.Int64(0),
		}).Return(nil, fmt.Errorf("someError")).Once()

	type fields struct {
		tickerDuration time.Duration
		cluster        string
		ecs            ecsiface.ECSAPI
	}
	type args struct {
		ctx           context.Context
		servicesRegex string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				tickerDuration: 10 * time.Millisecond,
				cluster:        "someCluster",
				ecs:            &mockECS,
			},
			args: args{
				ctx:           context.Background(),
				servicesRegex: "someArn",
			},
		},
		{
			name: "findServiceArns error",
			fields: fields{
				tickerDuration: 10 * time.Millisecond,
				cluster:        "someCluster",
				ecs:            &findServiceArnsErrECS,
			},
			args: args{
				ctx:           context.Background(),
				servicesRegex: "someArn",
			},
			wantErr: true,
		},
		{
			name: "updateServices error",
			fields: fields{
				tickerDuration: 10 * time.Millisecond,
				cluster:        "someCluster",
				ecs:            &updateServicesErrECS,
			},
			args: args{
				ctx:           context.Background(),
				servicesRegex: "someArn",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := serviceScaler{
				tickerDuration: tt.fields.tickerDuration,
				cluster:        tt.fields.cluster,
				ecs:            tt.fields.ecs,
			}
			if err := ss.scaleServices(tt.args.ctx, tt.args.servicesRegex); (err != nil) != tt.wantErr {
				t.Errorf("serviceScaler.scaleServices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
