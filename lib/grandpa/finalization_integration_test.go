// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package grandpa

import (
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func Test_FinalizationHandler_waitServices(t *testing.T) {
	t.Parallel()

	mockingErr := errors.New("mocked error")

	tests := map[string]struct {
		callHandlerStop           bool
		createFinalizationHandler func(*gomock.Controller) *finalizationHandler
		wantErr                   error
	}{
		"voting_round_finalisation_engine_finishes_successfully": {
			createFinalizationHandler: func(ctrl *gomock.Controller) *finalizationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Run().DoAndReturn(func() error {
						time.Sleep(3 * time.Second)
						return nil
					})

					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Run().DoAndReturn(func() error {
						time.Sleep(4 * time.Second)
						return nil
					})

					return mockEngine, mockVoting
				}

				return &finalizationHandler{
					newServices: builder,
					stopCh:      make(chan struct{}),
					handlerDone: make(chan struct{}),
					errorCh:     make(chan error),
				}
			},
		},

		"voting_round_fails_should_stop_engine_service": {
			wantErr: mockingErr,
			createFinalizationHandler: func(ctrl *gomock.Controller) *finalizationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					const failTime = 2 * time.Second

					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Run().DoAndReturn(func() error {
						time.Sleep(failTime)
						return mockingErr
					})
					mockVoting.EXPECT().Stop().Return(nil)

					// once the voting round fails the finalisation handler
					// should be awere of the error and call the stop method from
					// the engine which will release the start method from engine service
					engineStopCh := make(chan struct{})
					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Run().DoAndReturn(func() error {
						<-engineStopCh
						return nil
					})
					mockEngine.EXPECT().Stop().DoAndReturn(func() error {
						close(engineStopCh)
						return nil
					})

					return mockEngine, mockVoting
				}

				return &finalizationHandler{
					newServices: builder,
					stopCh:      make(chan struct{}),
					handlerDone: make(chan struct{}),
					errorCh:     make(chan error),
				}
			},
		},

		"engine_fails_should_stop_voting_round_service": {
			wantErr: mockingErr,
			createFinalizationHandler: func(ctrl *gomock.Controller) *finalizationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					const failTime = 2 * time.Second

					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Run().DoAndReturn(func() error {
						time.Sleep(failTime)
						return mockingErr
					})
					mockEngine.EXPECT().Stop().Return(nil)

					// once the finalisation engine fails the finalisation handler
					// should be awere of the error and call the stop method from the
					// voting round which will release the start method from voting round service
					votingStopChannel := make(chan struct{})
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Run().DoAndReturn(func() error {
						<-votingStopChannel
						return nil
					})
					mockVoting.EXPECT().Stop().DoAndReturn(func() error {
						close(votingStopChannel)
						return nil
					})

					return mockEngine, mockVoting
				}

				return &finalizationHandler{
					newServices: builder,
					stopCh:      make(chan struct{}),
					handlerDone: make(chan struct{}),
					errorCh:     make(chan error),
				}
			},
		},
	}

	for tname, tt := range tests {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			finalizationHandler := tt.createFinalizationHandler(ctrl)

			err := finalizationHandler.waitServices()
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func Test_FinalizationHandler_Stop_ShouldHalt_Services(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		sentinelErr error
		wantErr     string
		newHandler  func(t *testing.T) *finalizationHandler
	}{
		"halt_ephemeral_services_after_calling_stop": {
			// when we start the finalisation handler we instantiate
			// and call the Run method from each ephemeral services
			// (votingHandler, finalizationEngine) since they are mocked
			// they will wait until the Stop method being called to release
			// the blocking channel and return from the function
			newHandler: func(t *testing.T) *finalizationHandler {
				ctrl := gomock.NewController(t)
				builder := func() (engine ephemeralService, voting ephemeralService) {
					engineStopCh := make(chan struct{})
					mockEngine := NewMockephemeralService(ctrl)

					mockEngine.EXPECT().Run().DoAndReturn(func() error {
						<-engineStopCh
						return nil
					})
					mockEngine.EXPECT().Stop().DoAndReturn(func() error {
						close(engineStopCh)
						return nil
					})

					votingStopCh := make(chan struct{})
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Run().DoAndReturn(func() error {
						<-votingStopCh
						return nil
					})
					mockVoting.EXPECT().Stop().DoAndReturn(func() error {
						close(votingStopCh)
						return nil
					})
					return mockEngine, mockVoting
				}

				return &finalizationHandler{
					newServices: builder,
					// mocked initiate round function
					initiateRound: func() error { return nil },
					stopCh:        make(chan struct{}),
					handlerDone:   make(chan struct{}),
					errorCh:       make(chan error),
				}
			},
		},
		"halt_fails_to_stop_one_ephemeral_service": {
			sentinelErr: errServicesStopFailed,
			wantErr:     "services stop failed: cannot stop finalisation engine test",
			newHandler: func(t *testing.T) *finalizationHandler {
				ctrl := gomock.NewController(t)
				builder := func() (engine ephemeralService, voting ephemeralService) {
					engineStopCh := make(chan struct{})
					mockEngine := NewMockephemeralService(ctrl)

					mockEngine.EXPECT().Run().DoAndReturn(func() error {
						<-engineStopCh
						return nil
					})
					mockEngine.EXPECT().Stop().DoAndReturn(func() error {
						close(engineStopCh)
						return errors.New("cannot stop finalisation engine test")
					})

					votingStopCh := make(chan struct{})
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Run().DoAndReturn(func() error {
						<-votingStopCh
						return nil
					})
					mockVoting.EXPECT().Stop().DoAndReturn(func() error {
						close(votingStopCh)
						return nil
					})

					return mockEngine, mockVoting
				}

				return &finalizationHandler{
					newServices: builder,
					// mocked initiate round function
					initiateRound: func() error { return nil },
					stopCh:        make(chan struct{}),
					handlerDone:   make(chan struct{}),
					errorCh:       make(chan error),
				}
			},
		},

		"halt_fails_to_stop_both_ephemeral_service": {
			sentinelErr: errServicesStopFailed,
			wantErr: "services stop failed: cannot stop finalisation engine test; " +
				"cannot stop voting handler test",
			newHandler: func(t *testing.T) *finalizationHandler {
				ctrl := gomock.NewController(t)

				builder := func() (engine ephemeralService, voting ephemeralService) {
					engineStopCh := make(chan struct{})
					mockEngine := NewMockephemeralService(ctrl)

					mockEngine.EXPECT().Run().DoAndReturn(func() error {
						<-engineStopCh
						return nil
					})
					mockEngine.EXPECT().Stop().DoAndReturn(func() error {
						close(engineStopCh)
						return errors.New("cannot stop finalisation engine test")
					})

					votingStopCh := make(chan struct{})
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Run().DoAndReturn(func() error {
						<-votingStopCh
						return nil
					})
					mockVoting.EXPECT().Stop().DoAndReturn(func() error {
						close(votingStopCh)
						return errors.New("cannot stop voting handler test")
					})

					return mockEngine, mockVoting
				}

				return &finalizationHandler{
					newServices: builder,
					// mocked initiate round function
					initiateRound: func() error { return nil },
					stopCh:        make(chan struct{}),
					handlerDone:   make(chan struct{}),
					errorCh:       make(chan error),
				}
			},
		},
	}

	for tname, tt := range testcases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			handler := tt.newHandler(t)

			errorCh, err := handler.Start()
			require.NoError(t, err)

			// wait enough time to start subservices
			// and then call stop
			time.Sleep(2 * time.Second)
			err = handler.Stop()
			require.ErrorIs(t, err, tt.sentinelErr)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			}

			// since we are stopping the finalisation handler we expect
			// the errorCh to be closed without any error
			err, ok := <-errorCh
			require.False(t, ok,
				"expected channel to be closed, got an unexpected error: %s", err)
		})
	}
}
