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

func ProvideInherentData(
	blockState blockState,
	overseerCh chan<- any,
	relayParent common.Hash,
	inherentData *types.InherentData,
) error {
	parachainInherentData, err := createInherentData(blockState, overseerCh, relayParent)
	if err != nil {
		return fmt.Errorf("creating parachain inherent data: %w", err)
	}

	if inherentData == nil {
		inherentData = types.NewInherentData()
	}

	err = inherentData.SetInherent(types.Parachn0, parachainInherentData)
	if err != nil {
		return fmt.Errorf("setting parachain inherent data: %w", err)
	}

	return nil
}

func createInherentData(
	blockState blockState,
	overseerCh chan<- any,
	relayParent common.Hash,
) (*InherentData, error) {
	var (
		parentHeader        *types.Header
		provisionerInherent *provisioner.ProvisionerInherentData
		headerErr, provErr  error
		wg                  sync.WaitGroup
	)

	wg.Add(2)

	// Fetch parent header concurrently
	go func() {
		defer wg.Done()
		h, err := blockState.GetHeader(relayParent)
		if err != nil {
			headerErr = fmt.Errorf("getting header for relay parent %s: %w", relayParent, err)
			return
		}
		if h == nil {
			headerErr = fmt.Errorf("header for relay parent %s not found", relayParent)
			return
		}
		parentHeader = h
	}()

	// Fetch provisioner inherent data concurrently
	go func() {
		defer wg.Done()
		provisionerInherent, provErr = waitAndRequestProvisionerInherent(overseerCh, relayParent)
	}()

	wg.Wait()

	if headerErr != nil {
		return nil, headerErr
	}

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
