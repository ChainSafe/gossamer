package parachaininteraction

/*
The Statement Distribution Subsystem is responsible for distributing statements about seconded candidates between validators.
*/

// /home/kishan/code/polkadot/node/network/statement-distribution/src/lib.rs
// TODO: translate statement distribution message to go
// https://spec.polkadot.network/chapter-anv#net-msg-statement-distribution

// pub enum StatementDistributionMessage {
// 	/// A signed full statement under a given relay-parent.
// 	#[codec(index = 0)]
// 	Statement(Hash, UncheckedSignedFullStatement),
// 	/// Seconded statement with large payload (e.g. containing a runtime upgrade).
// 	///
// 	/// We only gossip the hash in that case, actual payloads can be fetched from sending node
// 	/// via request/response.
// 	#[codec(index = 1)]
// 	LargeStatement(StatementMetadata),
// }

// TODO: create statement message
// /// Create a network message from a given statement.
// fn statement_message(
// 	relay_parent: Hash,
// 	statement: SignedFullStatement,
// 	metrics: &Metrics,
// ) -> net_protocol::VersionedValidationProtocol {

// TODO: Circulate statement
// async fn circulate_statement<'a, Context>(
