package types

// ScheduledChangeType identifies a ScheduledChange consensus digest
var ScheduledChangeType = byte(1)

// ForcedChangeType identifies a ForcedChange consensus digest
var ForcedChangeType = byte(2)

// OnDisabledType identifies a DisabledChange consensus digest
var OnDisabledType = byte(3)

// PauseType identifies a Pause consensus digest
var PauseType = byte(4)

// ResumeType identifies a Resume consensus digest
var ResumeType = byte(5)

// GrandpaScheduledChange represents a GRANDPA scheduled authority change
type GrandpaScheduledChange struct {
	Auths []*GrandpaAuthorityDataRaw
	Delay uint32
}

// GrandpaForcedChange represents a GRANDPA forced authority change
type GrandpaForcedChange struct {
	Auths []*GrandpaAuthorityDataRaw
	Delay uint32
}

// OnDisabled represents a GRANDPA authority being disabled
type OnDisabled struct {
	ID uint64
}

// Pause represents an authority set pause
type Pause struct {
	Delay uint32
}

// Resume represents an authority set resume
type Resume struct {
	Delay uint32
}
