package parachain

import (
	"fmt"
	"sync"
	"time"

	"github.com/ChainSafe/gossamer/internal/log"

	provisioner "github.com/ChainSafe/gossamer/dot/parachain/provisioner/messages"
	parachaintypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
)

var logger = log.NewFromGlobal(log.AddContext("pkg", "parachain"))

var errResponseChannelClosed = fmt.Errorf("response channel is closed")

// InherentData represents parachains inherent-data passed into the runtime by a block author
type InherentData struct {
	// Bitfields represents signed bitfields by validators about availability.
	Bitfields []parachaintypes.UncheckedSignedAvailabilityBitfield `scale:"1"`
	// BackedCandidates represents backed candidates for inclusion in the block.
	BackedCandidates []parachaintypes.BackedCandidate `scale:"2"`
	// Disputes represents sets of dispute votes for inclusion.
	Disputes []parachaintypes.DisputeStatementSet `scale:"3"`
	// ParentHeader represents the parent block header. Used for checking state proofs.
	ParentHeader types.Header `scale:"4"`
}

type blockState interface {
	GetHeader(common.Hash) (*types.Header, error)
}

// ProvideInherentData creates the inherent data for the parachain block authoring process
// and returns it as a types.InherentData object with the parachain inherent data.
func ProvideInherentData(
	blockState blockState,
	overseerCh chan<- any,
	relayParent common.Hash,
) (*types.InherentData, error) {
	parachainInherentData, err := createInherentData(blockState, overseerCh, relayParent)
	if err != nil {
		return nil, fmt.Errorf("creating parachain inherent data: %w", err)
	}

	inherentData := types.NewInherentData()
	err = inherentData.SetInherent(types.Parachn0, parachainInherentData)
	if err != nil {
		return nil, fmt.Errorf("setting parachain inherent data: %w", err)
	}

	return inherentData, nil
}

func createInherentData(
	blockState blockState,
	overseerCh chan<- any,
	relayParent common.Hash,
) (*InherentData, error) {
	var (
		provisionerInherent *provisioner.ProvisionerInherentData
		provErr             error
		wg                  sync.WaitGroup
	)

	wg.Add(1)
	// Fetch provisioner inherent data concurrently
	go func() {
		defer wg.Done()
		provisionerInherent, provErr = waitAndRequestProvisionerInherent(overseerCh, relayParent)
	}()

	parentHeader, err := blockState.GetHeader(relayParent)
	if err != nil {
		return nil, fmt.Errorf("getting header for relay parent %s: %w", relayParent, err)
	}

	if parentHeader == nil {
		return nil, fmt.Errorf("header for relay parent %s not found", relayParent)
	}

	wg.Wait() // wait for the provisioner inherent data to be fetched

	if provErr != nil {
		logger.Errorf("getting provisioner inherent data: %s\n", provErr)
		return &InherentData{ParentHeader: *parentHeader}, provErr
	}

	uncheckBitfields := make([]parachaintypes.UncheckedSignedAvailabilityBitfield, 0, len(provisionerInherent.Bitfields))
	for _, bitfield := range provisionerInherent.Bitfields {
		uncheckBitfields = append(uncheckBitfields, parachaintypes.UncheckedSignedAvailabilityBitfield(bitfield))
	}

	return &InherentData{
		Bitfields:        uncheckBitfields,
		BackedCandidates: provisionerInherent.BackedCandidates,
		Disputes:         provisionerInherent.Disputes,
		ParentHeader:     *parentHeader,
	}, nil
}

func waitAndRequestProvisionerInherent(
	overseerCh chan<- any,
	relayParent common.Hash,
) (*provisioner.ProvisionerInherentData, error) {
	waitForActivation := parachaintypes.WaitForActivation{
		RelayParent: relayParent,
		ResponseCh:  make(chan error),
	}

	overseerCh <- waitForActivation
	select {
	case err, ok := <-waitForActivation.ResponseCh:
		if !ok {
			return nil, errResponseChannelClosed
		}

		if err != nil {
			return nil, fmt.Errorf("waiting for leaf activation: %s", err)
		}
	case <-time.After(parachaintypes.SubsystemRequestTimeout):
		return nil, parachaintypes.ErrSubsystemRequestTimeout
	}

	requestProvisionerInherentData := provisioner.RequestInherentData{
		RelayParent:             relayParent,
		ProvisionerInherentData: make(chan provisioner.ProvisionerInherentData),
	}

	// requesting inherent data after having waited for leaf activation
	overseerCh <- requestProvisionerInherentData
	select {
	case provisionerInherentData, ok := <-requestProvisionerInherentData.ProvisionerInherentData:
		if !ok {
			return nil, errResponseChannelClosed
		}
		return &provisionerInherentData, nil
	case <-time.After(parachaintypes.SubsystemRequestTimeout):
		return nil, parachaintypes.ErrSubsystemRequestTimeout
	}
}
