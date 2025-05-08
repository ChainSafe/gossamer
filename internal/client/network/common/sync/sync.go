package sync

// / Sync operation mode.
type SyncMode interface {
	/// Returns `true` if `self` is [`Self::Warp`].
	IsWarp() bool
	/// Returns `true` if `self` is [`Self::LightState`].
	LightState() bool
}

type (
	// / Full block download and verification.
	SyncModeFull struct{}
	// / Download blocks and the latest state.
	SyncModeLightState struct {
		/// Skip state proof download and verification.
		SkipProofs bool
		/// Download indexed transactions for recent blocks.
		StorageChainMode bool
	}
	// / Warp sync - verify authority set transitions and the latest state.
	SyncModeWarp struct{}
)

func (smf SyncModeFull) IsWarp() bool {
	return false
}
func (smls SyncModeLightState) IsWarp() bool {
	return false
}
func (smw SyncModeWarp) IsWarp() bool {
	return true
}

func (smf SyncModeFull) LightState() bool {
	return false
}
func (smls SyncModeLightState) LightState() bool {
	return true
}
func (smw SyncModeWarp) LightState() bool {
	return false
}

// impl Default for SyncMode {
// 	fn default() -> Self {
// 		Self::Full
// 	}
// }
