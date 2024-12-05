## Availability Recovery Subsystem Design

The design for the Availability Recovery Subsystem is based on the Polkadot Host specification as detailed in the
[Polkadot Parachain Host Implementers' Guide](https://paritytech.github.io/polkadot-sdk/book/node/availability/availability-recovery.html).
This subsystem is responsible for recovering the data made available via the Availability Distribution subsystem,
neccessary for candidate validation during the approval/disputes processes. Additionally, it is also being used by
collators to recover PoVs in adversarial scenarios where the other collators of the para are censoring blocks. The
implementation is composed of the following components:

- **Create skeleton for Availability Recovery subsystem**: Create skeleton code for the implementation of this 
  subsystem utilizing the subsystem pattern that has been used in other gossamer subsystems.  The details for this 
  implementation is described in [issue #4297](https://github.com/ChainSafe/gossamer/issues/4297). 
- **Implement logic for determing and launching recovery strategy**: This component is responsible for determining 
  the recovery strategy to be used based on the system configuration, parameters and session info.  It launches the 
  `RecoveryTask` using the determined strategy. The details for this implementation is described in [issue #4298](https://github.com/ChainSafe/gossamer/issues/4298).
- **Implement code for FullFetch recovery strategy**: This component is responsible for implementing the logic for 
  the FullFetch recovery strategy. This strategy tries requesting the full available data from the validators in the 
  backing group to which the node is already connected. They are tried one by one in a random order. It is very 
  performant if there's enough network bandwidth and the backing group is not overloaded. The costly reed-solomon 
  reconstruction is not needed. See [issue #4299](https://github.com/ChainSafe/gossamer/issues/4299) for more details.
- **Implement code for FetchSystematicChunks recovery strategy**: This component is responsible for implementing the 
  logic for the FetchSystematicChunks recovery strategy.  Very similar to FetchChunks below but requests from the
  validators that hold the systematic chunks, so that we avoid reed-solomon reconstruction. Only possible if
  `node_features::FeatureIndex::AvailabilityChunkMapping` is enabled and the core_index is supplied (currently only for
  recoveries triggered by approval voting). See [issue #4300](https://github.com/ChainSafe/gossamer/issues/4300) for more details.
- **Implement code for FetchChunks recovery strategy**: This component is responsible for implementing the logic for 
  the FetchChunks recovery strategy.  The least performant strategy but also the most comprehensive one. It's the 
  only one that cannot fail under the byzantine threshold assumption, so it's always added as the last one in the 
  recovery_strategies queue. Performs parallel chunk requests to validators. When enough chunks were received, do 
  the reconstruction. In the worst case, all validators will be tried. See [issue #4301](https://github.com/ChainSafe/gossamer/issues/4301) for more details.
