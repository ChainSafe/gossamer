package columns

type column uint32

const (
	Meta column = 0
	/// maps hashes to lookup keys and numbers to canon hashes.
	KeyLookup      column = 3
	Header         column = 4
	Body           column = 5
	Justifications column = 6
	// pub const AUX: u32 = 8;
	// /// Offchain workers local storage
	// pub const OFFCHAIN: u32 = 9;
	// /// Transactions
	// pub const TRANSACTION: u32 = 11;
	// pub const BODY_INDEX: u32 = 12;
	BodyIndex column = 12
)
