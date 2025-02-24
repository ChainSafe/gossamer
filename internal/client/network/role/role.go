package role

// Role that the peer sent to us during the handshake, with the addition of what our local node knows about that peer.
//
// This type is  different from the [Role]. The [Role] type indicates what a node says about itself, while ObservedRole
// is a [Role] merged with the information known locally about that node.
type ObservedRole uint

const (
	// Full node.
	ObservedRoleFull ObservedRole = iota
	// Light node.
	ObservedRoleLight
	// Third-party authority.
	ObservedRoleAuthority
)

// Role of the local node.
type Role uint

const (
	// Regular full node.
	RoleFull Role = iota
	// Actual authority.
	RoleAuthority
)

// Roles are a bitmask of the roles that a node fulfills.
type Roles uint8

const (
	// No network.
	RolesNone Roles = 0b00000000
	// Full node, does not participate in consensus.
	RolesFull Roles = 0b00000001
	// Light client node.
	RolesLight Roles = 0b00000010
	// Act as an authority
	RolesAuthority Roles = 0b00000100
)

func (r Roles) intersects(other Roles) bool {
	return !((r & other) == 0)
}

// Does this role represents a client that holds full chain data locally?
func (r Roles) IsFull() bool {
	return r.intersects(RolesFull | RolesAuthority)
}

// Does this role represents a client that does not participates in the consensus?
func (r Roles) IsAuthority() bool {
	return r == RolesAuthority
}

func (r Roles) ObservedRole() ObservedRole {
	if r.IsAuthority() {
		return ObservedRoleAuthority
	} else if r.IsFull() {
		return ObservedRoleFull
	} else {
		return ObservedRoleLight
	}
}
