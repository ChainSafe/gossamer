// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package metrics

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Server(t *testing.T) {
	t.Parallel()

	server := NewServer("127.0.0.1:0")

	err := server.Start()

	assert.NoError(t, err)

	url := "http://" + server.server.GetAddress() + "/metrics"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	require.NoError(t, err)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	err = response.Body.Close()
	require.NoError(t, err)
	assert.Contains(t, string(data), "# HELP")

	err = server.Stop()
	require.NoError(t, err)
}

func TestNewIntervalConfig(t *testing.T) {
	type args struct {
		publish bool
	}
	tests := []struct {
		name string
		args args
		want IntervalConfig
	}{
		{
			name: "happy path",
			args: args{
				publish: true,
			},
			want: IntervalConfig{
				Publish:  true,
				Interval: defaultInterval,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewIntervalConfig(tt.args.publish); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewIntervalConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}
