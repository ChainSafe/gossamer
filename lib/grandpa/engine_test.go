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
					servicesBuilder: builder,
					timeoutStop:     2 * time.Second,
					observableErrs:  make(chan error),
					stopCh:          make(chan struct{}),
					handlerDone:     make(chan struct{}),
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
					servicesBuilder: builder,
					timeoutStop:     2 * time.Second,
					observableErrs:  make(chan error),
					stopCh:          make(chan struct{}),
					handlerDone:     make(chan struct{}),
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
						timeToFail := time.NewTimer(failTime)
						<-timeToFail.C
						return errors.New("mocked voting round fails")
					})

					// once the voting round fails the finalization handler
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
					servicesBuilder: builder,
					timeoutStop:     2 * time.Second,
					observableErrs:  make(chan error),
					stopCh:          make(chan struct{}),
					handlerDone:     make(chan struct{}),
				}
			},
		},

		"engine_fails_should_stop_voting_round_service": {
			wantErr: errors.New("mocked finalization engine fails"),
			createFinalizationHandler: func(ctrl *gomock.Controller) *finalizationHandler {
				builder := func() (engine ephemeralService, voting ephemeralService) {
					failTime := 2 * time.Second

					mockEngine := NewMockephemeralService(ctrl)
					mockEngine.EXPECT().Start().DoAndReturn(func() error {
						timeToFail := time.NewTimer(failTime)
						<-timeToFail.C
						return errors.New("mocked finalization engine fails")
					})

					// once the finalization engine fails the finalization handler
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
					servicesBuilder: builder,
					timeoutStop:     2 * time.Second,
					observableErrs:  make(chan error),
					stopCh:          make(chan struct{}),
					handlerDone:     make(chan struct{}),
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

	handler := &finalizationHandler{
		servicesBuilder: builder,
		// mocked initiate round function
		initiateRound:  func() error { return nil },
		timeoutStop:    2 * time.Second,
		observableErrs: make(chan error),
		stopCh:         make(chan struct{}),
		handlerDone:    make(chan struct{}),
	}

	doneCh := make(chan struct{})
	errsCh, err := handler.Start()
	require.NoError(t, err)

	go func() {
		defer close(doneCh)
		for err := range errsCh {
			t.Errorf("expected no error, got %s", err)
			return
		}
	}()

	// wait enough time to start subservices
	time.After(2 * time.Second)
	err = handler.Stop()
	require.NoError(t, err)

	<-doneCh
}
