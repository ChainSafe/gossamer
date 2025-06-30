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

type blockState interface {
	GetHeader(common.Hash) (*types.Header, error)
}

// CreateInherentData fetches the necessary data for the parachain inherent and returns it as an InherentData object.
func CreateInherentData(
	blockState blockState,
	overseerCh chan<- any,
	relayParent common.Hash,
) (*parachaintypes.InherentData, error) {
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
		return &parachaintypes.InherentData{ParentHeader: *parentHeader}, provErr
	}

	uncheckBitfields := make([]parachaintypes.UncheckedSignedAvailabilityBitfield, 0, len(provisionerInherent.Bitfields))
	for _, bitfield := range provisionerInherent.Bitfields {
		uncheckBitfields = append(uncheckBitfields, parachaintypes.UncheckedSignedAvailabilityBitfield(bitfield))
	}

	return &parachaintypes.InherentData{
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
