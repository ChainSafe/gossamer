package grandpa

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
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
				}
			},
		},

		"engine_fails_should_stop_voting_round_service": {
			errString: "finalisation engine ephemeral failed: mocked finalization engine failed",
			wantErr:   errFinalizationEngineFailed,
			createfinalisationHandler: func(ctrl *gomock.Controller) *finalisationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Run().DoAndReturn(func() error {
						return errors.New("mocked finalization engine failed")
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

			err := finalisationHandler.runEphemeralServices()

			require.ErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				require.EqualError(t, err, tt.errString)
			}
		})
	}
}
