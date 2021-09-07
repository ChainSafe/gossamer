package sync

var _ workHandler = &bootstrapSyncer{}

type bootstrapSyncer struct {
}

func newBootStrapSyncer() *bootstrapSyncer {
	return &bootstrapSyncer{}
}

func (s *bootstrapSyncer) handleWork(*peerState)      {}
func (s *bootstrapSyncer) handleWorkerResult(*worker) {}
func (s *bootstrapSyncer) handleTick()                {}
