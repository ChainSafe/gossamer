// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/devnet/cmd/scale-down-ecs-service/internal"
	"github.com/aws/aws-sdk-go/aws"
	request "github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
	"go.uber.org/mock/gomock"
)

func Test_serviceScaler_findServiceArns(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockECS := NewMockECSAPI(ctrl)
	mockECS.EXPECT().
		ListServicesWithContext(gomock.Any(), &ecs.ListServicesInput{
			Cluster: aws.String("someCluster"),
		}).Return(&ecs.ListServicesOutput{
		ServiceArns: []*string{
			aws.String("someArn0"),
			aws.String("someArn1"),
		},
		NextToken: aws.String("someNextToken")}, nil)
	mockECS.EXPECT().
		ListServicesWithContext(gomock.Any(), &ecs.ListServicesInput{
			Cluster:   aws.String("someCluster"),
			NextToken: aws.String("someNextToken"),
		}).Return(&ecs.ListServicesOutput{
		ServiceArns: []*string{
			aws.String("someArn2"),
			aws.String("someArn3"),
		}}, nil)
	mockECS.EXPECT().
		ListServicesWithContext(gomock.Any(), &ecs.ListServicesInput{
			Cluster: aws.String("someErrCluster"),
		}).Return(nil, fmt.Errorf("someErr"))
	mockECS.EXPECT().
		ListServicesWithContext(gomock.Any(), &ecs.ListServicesInput{
			Cluster: aws.String("someEmptyCluster"),
		}).Return(&ecs.ListServicesOutput{}, nil)

	type fields struct {
		tickerDuration time.Duration
		cluster        string
		ecs            internal.ECSAPI
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
			name: "with_next_token",
			fields: fields{
				cluster: "someCluster",
				ecs:     mockECS,
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
			name: "ListServicesWithContext_err",
			fields: fields{
				cluster: "someErrCluster",
				ecs:     mockECS,
			},
			args: args{
				ctx:          context.Background(),
				serviceRegex: "someArn",
			},
			wantErr: true,
		},
		{
			name: "no_services_err",
			fields: fields{
				cluster: "someEmptyCluster",
				ecs:     mockECS,
			},
			args: args{
				ctx:          context.Background(),
				serviceRegex: "someArn",
			},
			wantErr: true,
		},
		{
			name: "regex_err",
			fields: fields{
				ecs: mockECS,
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

func Test_serviceScaler_drainServices(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockECS := NewMockECSAPI(ctrl)
	mockECS.EXPECT().
		UpdateServiceWithContext(gomock.Any(), &ecs.UpdateServiceInput{
			Cluster:      aws.String("someCluster"),
			Service:      aws.String("someArn0"),
			DesiredCount: aws.Int64(0),
		}).Return(&ecs.UpdateServiceOutput{}, nil)
	mockECS.EXPECT().
		UpdateServiceWithContext(gomock.Any(), &ecs.UpdateServiceInput{
			Cluster:      aws.String("someCluster"),
			Service:      aws.String("someArn1"),
			DesiredCount: aws.Int64(0),
		}).Return(&ecs.UpdateServiceOutput{}, nil)
	mockECS.EXPECT().
		UpdateServiceWithContext(gomock.Any(), &ecs.UpdateServiceInput{
			Cluster:      aws.String("someErrCluster"),
			Service:      aws.String("someArn0"),
			DesiredCount: aws.Int64(0),
		}).Return(nil, fmt.Errorf("some Error"))

	type fields struct {
		tickerDuration time.Duration
		cluster        string
		ecs            internal.ECSAPI
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
			name: "happy_path",
			fields: fields{
				cluster: "someCluster",
				ecs:     mockECS,
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
			name: "UpdateServiceWithContext_err",
			fields: fields{
				cluster: "someErrCluster",
				ecs:     mockECS,
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
			if err := ss.drainServices(tt.args.ctx, tt.args.serviceArns); (err != nil) != tt.wantErr {
				t.Errorf("serviceScaler.drainServices() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_serviceScaler_waitForRunningCount(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockECS := NewMockECSAPI(ctrl)
	mockECS.EXPECT().
		DescribeServicesWithContext(gomock.Any(), &ecs.DescribeServicesInput{
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
		}}, nil)
	mockECS.EXPECT().
		DescribeServicesWithContext(gomock.Any(), &ecs.DescribeServicesInput{
			Cluster: aws.String("someErrorCluster"),
			Services: []*string{
				aws.String("someArn0"),
				aws.String("someArn1"),
			},
		}).Return(nil, fmt.Errorf("someError"))

	ctx, cancel := context.WithCancel(context.Background())
	mockECSCancel := NewMockECSAPI(ctrl)
	mockECSCancel.EXPECT().
		DescribeServicesWithContext(gomock.Any(), &ecs.DescribeServicesInput{
			Cluster: aws.String("someCluster"),
			Services: []*string{
				aws.String("someArn0"),
				aws.String("someArn1"),
			},
		}).DoAndReturn(func(_ context.Context, _ *ecs.DescribeServicesInput, _ ...request.Option) (
		*ecs.DescribeServicesOutput, error) {
		go func() {
			// should trigger before 10ms ticker
			<-time.After(2 * time.Millisecond)
			cancel()
		}()

		return &ecs.DescribeServicesOutput{
			Services: []*ecs.Service{
				{
					RunningCount: aws.Int64(1),
					ServiceArn:   aws.String("someArn0"),
				},
				{
					RunningCount: aws.Int64(1),
					ServiceArn:   aws.String("someArn1"),
				},
			}}, nil
	})

	type fields struct {
		tickerDuration time.Duration
		cluster        string
		ecs            internal.ECSAPI
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
			name: "already_0_zero_running_count",
			fields: fields{
				tickerDuration: time.Nanosecond,
				cluster:        "someCluster",
				ecs:            mockECS,
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
			name: "DescribeServicesWithContext_error",
			fields: fields{
				tickerDuration: time.Nanosecond,
				cluster:        "someErrorCluster",
				ecs:            mockECS,
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
			name: "context_cancel_err",
			fields: fields{
				tickerDuration: 10 * time.Millisecond,
				cluster:        "someCluster",
				ecs:            mockECSCancel,
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
		ecs            internal.ECSAPI
	}
	tests := []struct {
		name string
		args args
		want *serviceScaler
	}{
		{
			name: "already_0_zero_running_count",
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
	ctrl := gomock.NewController(t)

	mockECS := NewMockECSAPI(ctrl)
	mockECS.EXPECT().
		ListServicesWithContext(gomock.Any(), &ecs.ListServicesInput{
			Cluster: aws.String("someCluster"),
		}).Return(&ecs.ListServicesOutput{
		ServiceArns: []*string{
			aws.String("someArn0"),
			aws.String("someArn1"),
		}}, nil)
	mockECS.EXPECT().
		UpdateServiceWithContext(gomock.Any(), &ecs.UpdateServiceInput{
			Cluster:      aws.String("someCluster"),
			Service:      aws.String("someArn0"),
			DesiredCount: aws.Int64(0),
		}).Return(&ecs.UpdateServiceOutput{}, nil)
	mockECS.EXPECT().
		UpdateServiceWithContext(gomock.Any(), &ecs.UpdateServiceInput{
			Cluster:      aws.String("someCluster"),
			Service:      aws.String("someArn1"),
			DesiredCount: aws.Int64(0),
		}).Return(&ecs.UpdateServiceOutput{}, nil)
	mockECS.EXPECT().
		DescribeServicesWithContext(gomock.Any(), &ecs.DescribeServicesInput{
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
		}}, nil)

	findServiceArnsErrECS := NewMockECSAPI(ctrl)
	findServiceArnsErrECS.EXPECT().
		ListServicesWithContext(gomock.Any(), &ecs.ListServicesInput{
			Cluster: aws.String("someCluster"),
		}).Return(nil, fmt.Errorf("someError"))

	updateServicesErrECS := NewMockECSAPI(ctrl)
	updateServicesErrECS.EXPECT().
		ListServicesWithContext(gomock.Any(), &ecs.ListServicesInput{
			Cluster: aws.String("someCluster"),
		}).Return(&ecs.ListServicesOutput{
		ServiceArns: []*string{
			aws.String("someArn0"),
			aws.String("someArn1"),
		}}, nil)
	updateServicesErrECS.EXPECT().
		UpdateServiceWithContext(gomock.Any(), &ecs.UpdateServiceInput{
			Cluster:      aws.String("someCluster"),
			Service:      aws.String("someArn0"),
			DesiredCount: aws.Int64(0),
		}).Return(nil, fmt.Errorf("someError"))

	type fields struct {
		tickerDuration time.Duration
		cluster        string
		ecs            internal.ECSAPI
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
			name: "happy_path",
			fields: fields{
				tickerDuration: 10 * time.Millisecond,
				cluster:        "someCluster",
				ecs:            mockECS,
			},
			args: args{
				ctx:           context.Background(),
				servicesRegex: "someArn",
			},
		},
		{
			name: "findServiceArns_error",
			fields: fields{
				tickerDuration: 10 * time.Millisecond,
				cluster:        "someCluster",
				ecs:            findServiceArnsErrECS,
			},
			args: args{
				ctx:           context.Background(),
				servicesRegex: "someArn",
			},
			wantErr: true,
		},
		{
			name: "updateServices_error",
			fields: fields{
				tickerDuration: 10 * time.Millisecond,
				cluster:        "someCluster",
				ecs:            updateServicesErrECS,
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
