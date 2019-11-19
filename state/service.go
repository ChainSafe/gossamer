package state

type Service struct {
	storage storageState
	block blockState
	net networkState

}

func NewService() *Service {
	return &Service{
		storage: storageState{},
		block:   blockState{},
		net:     networkState{},
	}
}

func (s *Service) Start() error {

}

func (s *Service) Stop() error {

}

