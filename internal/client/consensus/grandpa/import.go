package grandpa

// / A block-import handler for GRANDPA.
// /
// / This scans each imported block for signals of changing authority set.
// / If the block being imported enacts an authority set change then:
// / - If the current authority set is still live: we import the block
// / - Otherwise, the block must include a valid justification.
// /
// / When using GRANDPA, the block import worker should be using this block import
// / object.im
// pub struct GrandpaBlockImport<Backend, Block: BlockT, Client, SC> {
type GrandpaBlockImport struct {
	// inner: Arc<Client>,
	// justification_import_period: u32,
	// select_chain: SC,
	// authority_set: SharedAuthoritySet<Block::Hash, NumberFor<Block>>,
	// send_voter_commands: TracingUnboundedSender<VoterCommand<Block::Hash, NumberFor<Block>>>,
	// authority_set_hard_forks:
	//
	//	Mutex<HashMap<Block::Hash, PendingChange<Block::Hash, NumberFor<Block>>>>,
	//
	// justification_sender: GrandpaJustificationSender<Block>,
	// telemetry: Option<TelemetryHandle>,
	// _phantom: PhantomData<Backend>,
}

// impl<Backend, Block: BlockT, Client, SC: Clone> Clone
// 	for GrandpaBlockImport<Backend, Block, Client, SC>
// {
// 	fn clone(&self) -> Self {
// 		GrandpaBlockImport {
// 			inner: self.inner.clone(),
// 			justification_import_period: self.justification_import_period,
// 			select_chain: self.select_chain.clone(),
// 			authority_set: self.authority_set.clone(),
// 			send_voter_commands: self.send_voter_commands.clone(),
// 			authority_set_hard_forks: Mutex::new(self.authority_set_hard_forks.lock().clone()),
// 			justification_sender: self.justification_sender.clone(),
// 			telemetry: self.telemetry.clone(),
// 			_phantom: PhantomData,
// 		}
// 	}
// }
