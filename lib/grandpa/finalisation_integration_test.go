// Copyright 2022 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

//go:build integration

package grandpa

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func Test_finalisationHandler_runEphemeralServices(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		callHandlerStop           bool
		createfinalisationHandler func(*gomock.Controller) *finalisationHandler
		wantErr                   error
		errString                 string
	}{
		"voting_round_finalisation_engine_finishes_successfully": {
			createfinalisationHandler: func(ctrl *gomock.Controller) *finalisationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Run().DoAndReturn(func() error {
						return nil
					})

					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Run().DoAndReturn(func() error {
						return nil
					})

					return mockEngine, mockVoting
				}

				return &finalisationHandler{
					newServices: builder,
					stopCh:      make(chan struct{}),
					handlerDone: make(chan struct{}),
					firstRun:    false,
				}
			},
		},

		"voting_round_fails_should_stop_engine_service": {
			errString: "voting round ephemeral failed: mocked voting round failed",
			wantErr:   errvotingRoundHandlerFailed,
			createfinalisationHandler: func(ctrl *gomock.Controller) *finalisationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					mockVoting := NewMockephemeralService(ctrl)
					mockVoting.EXPECT().Run().DoAndReturn(func() error {
						time.Sleep(time.Second)
						return errors.New("mocked voting round failed")
					})

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

				return &finalisationHandler{
					newServices: builder,
					stopCh:      make(chan struct{}),
					handlerDone: make(chan struct{}),
					firstRun:    false,
				}
			},
		},

		"engine_fails_should_stop_voting_round_service": {
			errString: "finalisation engine ephemeral failed: mocked finalisation engine failed",
			wantErr:   errfinalisationEngineFailed,
			createfinalisationHandler: func(ctrl *gomock.Controller) *finalisationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Run().DoAndReturn(func() error {
						time.Sleep(time.Second)
						return errors.New("mocked finalisation engine failed")
					})

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

				return &finalisationHandler{
					newServices: builder,
					stopCh:      make(chan struct{}),
					handlerDone: make(chan struct{}),
					firstRun:    false,
				}
			},
		},
	}

	for tname, tt := range tests {
		tt := tt

		t.Run(tname, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			finalisationHandler := tt.createfinalisationHandler(ctrl)

			// passing the ready channel as nil since the first run is false
			// and we ensure the method fh.newServices() is being called
			err := finalisationHandler.runEphemeralServices(nil)
			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.errString)
			}
		})
	}
}

func Test_finalisationHandler_Stop_ShouldHalt_Services(t *testing.T) {
	t.Parallel()

	testcases := map[string]struct {
		wantErr    error
		errString  string
		newHandler func(*gomock.Controller) *finalisationHandler
	}{
		"halt_ephemeral_services_after_calling_stop": {
			// when we start the finalisation handler we instantiate
			// and call the Run method from each ephemeral services
			// (votingHandler, finalisationEngine) since they are mocked
			// they will wait until the Stop method being called to release
			// the blocking channel and return from the function
			newHandler: func(ctrl *gomock.Controller) *finalisationHandler {
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

				return &finalisationHandler{
					newServices: builder,
					// mocked initiate round function
					initiateRound: func() error { return nil },
					stopCh:        make(chan struct{}),
					handlerDone:   make(chan struct{}),
					firstRun:      true,
				}
			},
		},
		"halt_fails_to_stop_one_ephemeral_service": {
			wantErr:   errServicesStopFailed,
			errString: "services stop failed: cannot stop finalisation engine test",
			newHandler: func(ctrl *gomock.Controller) *finalisationHandler {
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

				return &finalisationHandler{
					newServices: builder,
					// mocked initiate round function
					initiateRound: func() error { return nil },
					stopCh:        make(chan struct{}),
					handlerDone:   make(chan struct{}),
					firstRun:      true,
				}
			},
		},
		"halt_fails_to_stop_both_ephemeral_service": {
			wantErr: errServicesStopFailed,
			errString: "services stop failed: cannot stop finalisation engine test; " +
				"cannot stop voting handler test",
			newHandler: func(ctrl *gomock.Controller) *finalisationHandler {
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

				return &finalisationHandler{
					newServices: builder,
					// mocked initiate round function
					initiateRound: func() error { return nil },
					stopCh:        make(chan struct{}),
					handlerDone:   make(chan struct{}),
					firstRun:      true,
				}
			},
		},
	}

	for tname, tt := range testcases {
		tt := tt
		t.Run(tname, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			handler := tt.newHandler(ctrl)

			errorCh, err := handler.Start()
			require.NoError(t, err)

			// wait enough time to start subservices
			// and then call stop
			time.Sleep(2 * time.Second)
			err = handler.Stop()
			require.ErrorIs(t, err, tt.wantErr)
			if tt.errString != "" {
				require.EqualError(t, err, tt.errString)
			}

			// since we are stopping the finalisation handler we expect
			// the errorCh to be closed without any error
			err, ok := <-errorCh
			require.Falsef(t, ok,
				"expected channel to be closed, got an unexpected error: %s", err)
		})
	}
}
