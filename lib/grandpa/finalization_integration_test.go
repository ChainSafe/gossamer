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

	tests := map[string]struct {
		callHandlerStop           bool
		createFinalizationHandler func(*gomock.Controller) *finalizationHandler
		wantErr                   error
	}{
		"voting_round_finishes_successfully": {
			createFinalizationHandler: func(ctrl *gomock.Controller) *finalizationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Start().DoAndReturn(func() error {
						<-time.NewTimer(3 * time.Second).C
						return nil
					})
					mockVoting.EXPECT().Stop().Return(nil)

					engineStopCh := make(chan struct{})
					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Start().DoAndReturn(func() error {
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
					timeout:     2 * time.Second,
					stopCh:      make(chan struct{}),
					handlerDone: make(chan struct{}),
					errorCh:     make(chan error),
				}
			},
		},

		"engine_finishes_successfully": {
			createFinalizationHandler: func(ctrl *gomock.Controller) *finalizationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Start().DoAndReturn(func() error {
						<-time.NewTimer(3 * time.Second).C
						return nil
					})
					mockEngine.EXPECT().Stop().Return(nil)

					votingStopCh := make(chan struct{})
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Start().DoAndReturn(func() error {
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
					timeout:     2 * time.Second,
					stopCh:      make(chan struct{}),
					handlerDone: make(chan struct{}),
					errorCh:     make(chan error),
				}
			},
		},

		"voting_round_fails_should_stop_engine_service": {
			wantErr: errors.New("mocked voting round fails"),
			createFinalizationHandler: func(ctrl *gomock.Controller) *finalizationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					failTime := 2 * time.Second

					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Start().DoAndReturn(func() error {
						time.Sleep(failTime)
						return errors.New("mocked voting round fails")
					})
					mockVoting.EXPECT().Stop().Return(nil)

					// once the voting round fails the finalisation handler
					// should be awere of the error and call the stop method from
					// the engine which will release the start method from engine service
					engineStopCh := make(chan struct{})
					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Start().DoAndReturn(func() error {
						select {
						case <-time.After(failTime + time.Second):
							return errors.New("timeout waiting engineStopCh")
						case <-engineStopCh:
							return nil
						}
					})
					mockEngine.EXPECT().Stop().DoAndReturn(func() error {
						close(engineStopCh)
						return nil
					})

					return mockEngine, mockVoting
				}

				return &finalizationHandler{
					newServices: builder,
					timeout:     2 * time.Second,
					stopCh:      make(chan struct{}),
					handlerDone: make(chan struct{}),
					errorCh:     make(chan error),
				}
			},
		},

		"engine_fails_should_stop_voting_round_service": {
			wantErr: errors.New("mocked finalisation engine fails"),
			createFinalizationHandler: func(ctrl *gomock.Controller) *finalizationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					failTime := 2 * time.Second

					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Start().DoAndReturn(func() error {
						time.Sleep(failTime)
						return errors.New("mocked finalisation engine fails")
					})
					mockEngine.EXPECT().Stop().Return(nil)

					// once the finalisation engine fails the finalisation handler
					// should be awere of the error and call the stop method from the
					// voting round which will release the start method from voting round service
					votingStopChannel := make(chan struct{})
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Start().DoAndReturn(func() error {
						select {
						case <-time.After(failTime + time.Second):
							return errors.New("timeout waiting votingStopChannel")
						case <-votingStopChannel:
							return nil
						}
					})
					mockVoting.EXPECT().Stop().DoAndReturn(func() error {
						close(votingStopChannel)
						return nil
					})

					return mockEngine, mockVoting
				}

				return &finalizationHandler{
					newServices: builder,
					timeout:     2 * time.Second,
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
			if tt.wantErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.wantErr.Error())
			} else {
				require.NoError(t, err)
			}
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
			// when we start the finalization handler we instantiate
			// and call the Start method from each ephemeral services
			// (votingHandler, finalizationEngine) since they are mocked
			// they will wait until the Stop method being called to release
			// the blocking channel and return from the function
			newHandler: func(t *testing.T) *finalizationHandler {
				ctrl := gomock.NewController(t)
				builder := func() (engine ephemeralService, voting ephemeralService) {
					engineStopCh := make(chan struct{})
					mockEngine := NewMockephemeralService(ctrl)

					mockEngine.EXPECT().Start().DoAndReturn(func() error {
						<-engineStopCh
						return nil
					})
					mockEngine.EXPECT().Stop().DoAndReturn(func() error {
						close(engineStopCh)
						return nil
					})

					votingStopCh := make(chan struct{})
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Start().DoAndReturn(func() error {
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
					timeout:       2 * time.Second,
					stopCh:        make(chan struct{}),
					handlerDone:   make(chan struct{}),
					errorCh:       make(chan error),
				}
			},
		},
		"halt_fails_to_stop_one_ephemeral_service": {
			sentinelErr: errServicesStopFailed,
			wantErr:     "services stop failed: cannot stop finalization engine test",
			newHandler: func(t *testing.T) *finalizationHandler {
				ctrl := gomock.NewController(t)
				builder := func() (engine ephemeralService, voting ephemeralService) {
					engineStopCh := make(chan struct{})
					mockEngine := NewMockephemeralService(ctrl)

					mockEngine.EXPECT().Start().DoAndReturn(func() error {
						<-engineStopCh
						return nil
					})
					mockEngine.EXPECT().Stop().DoAndReturn(func() error {
						close(engineStopCh)
						return errors.New("cannot stop finalization engine test")
					})

					votingStopCh := make(chan struct{})
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Start().DoAndReturn(func() error {
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
					timeout:       2 * time.Second,
					stopCh:        make(chan struct{}),
					handlerDone:   make(chan struct{}),
					errorCh:       make(chan error),
				}
			},
		},

		"halt_fails_to_stop_both_ephemeral_service": {
			sentinelErr: errServicesStopFailed,
			wantErr: "services stop failed: cannot stop finalization engine test; " +
				"cannot stop voting handler test",
			newHandler: func(t *testing.T) *finalizationHandler {
				ctrl := gomock.NewController(t)

				builder := func() (engine ephemeralService, voting ephemeralService) {
					engineStopCh := make(chan struct{})
					mockEngine := NewMockephemeralService(ctrl)

					mockEngine.EXPECT().Start().DoAndReturn(func() error {
						<-engineStopCh
						return nil
					})
					mockEngine.EXPECT().Stop().DoAndReturn(func() error {
						close(engineStopCh)
						return errors.New("cannot stop finalization engine test")
					})

					votingStopCh := make(chan struct{})
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Start().DoAndReturn(func() error {
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
					timeout:       2 * time.Second,
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
			handler := tt.newHandler(t)

			errorCh, err := handler.Start()

			// wait enough time to start subservices
			// and then call stop
			time.Sleep(2 * time.Second)
			err = handler.Stop()
			require.ErrorIs(t, err, tt.sentinelErr)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
			}

			// since we are stopping the finalization handler we expect
			// the errorCh to be closed without any error
			err, ok := <-errorCh
			require.False(t, ok,
				"expected channel to be closed, got an unexpected error: %s", err)
		})
	}
}
