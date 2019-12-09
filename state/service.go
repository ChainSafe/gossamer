package state

type Service struct {
	db_path string
	Storage *storageState
	Block   *blockState
	Net     *networkState
}

func NewService(path string) *Service {
	return &Service{
		db_path: path,
		Storage: &storageState{},
		Block:   &blockState{},
		Net:     &networkState{},
	}
}

func (s *Service) Start() error {
	s.Storage = NewStorageState()
	s.Block = NewBlockState()
	s.Net = NewNetworkState()

	return nil
}

func (s *Service) Stop() error {
	return nil
}
