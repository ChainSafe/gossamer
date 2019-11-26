package state

type Service struct {
	storage storageState
	block   blockState
	net     networkState
}

func NewService() *Service {
	return &Service{
		storage: storageState{},
		block:   blockState{},
		net:     networkState{},
	}
}

func (s *Service) Start() error {
	return nil
}

func (s *Service) Stop() error {
	return nil
}