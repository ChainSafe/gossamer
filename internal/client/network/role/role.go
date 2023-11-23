package role

// Role of the local node.
type Role uint

const (
	/// Regular full node.
	RoleFull Role = iota
	/// Actual authority.
	RoleAuthority
)

// / Role that the peer sent to us during the handshake, with the addition of what our local node
// / knows about that peer.
// /
// / > **Note**: This enum is different from the `Role` enum. The `Role` enum indicates what a
// / >			node says about itself, while `ObservedRole` is a `Role` merged with the
// / >			information known locally about that node.
// #[derive(Debug, Clone)]
//
//	pub enum ObservedRole {
//		/// Full node.
//		Full,
//		/// Light node.
//		Light,
//		/// Third-party authority.
//		Authority,
//	}
type ObservedRole uint

const (
	ObservedRoleFull ObservedRole = iota
	ObservedRoleLight
	ObservedRoleAuthority
)
