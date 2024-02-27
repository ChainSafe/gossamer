package columns

type column uint32

const (
	Meta      column = 0
	State     column = 1
	StateMeta column = 2
	/// maps hashes to lookup keys and numbers to canon hashes.
	KeyLookup      column = 3
	Header         column = 4
	Body           column = 5
	Justifications column = 6
	Aux            column = 8
	// /// Offchain workers local storage
	// pub const OFFCHAIN: u32 = 9;
	/// Transactions
	Transaction column = 11
	BodyIndex   column = 12
)
