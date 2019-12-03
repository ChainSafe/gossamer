package state

type Service struct {
	Storage *storageState
	Block   blockState
	Net     networkState
}

func NewService() *Service {
	return &Service{
		Storage: &storageState{},
		Block:   blockState{},
		Net:     networkState{},
	}
}

func (s *Service) Start() error {
	s.Storage = NewStorageState()
	s.Block = *newBlockState()
	s.Net = *newNetworkState()

	return nil
}

func (s *Service) Stop() error {
	return nil
}
