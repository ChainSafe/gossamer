package availability_store

func Register(overseerChan chan<- any) (*AvailabilityStore, error) {
	availabilityStore := AvailabilityStore{
		SubSystemToOverseer: overseerChan,
	}

	return &availabilityStore, nil
}
