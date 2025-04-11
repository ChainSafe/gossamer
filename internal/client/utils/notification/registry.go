package notification

// / The shared structure to keep track on subscribers.
// #[derive(Debug, Default)]
//
//	pub(super) struct Registry {
//		pub(super) subscribers: HashSet<SeqID>,
//	}
type registry[Payload any] struct {
	subscribers map[uint64]struct{}
}

//	impl Subscribe<()> for Registry {
//		fn subscribe(&mut self, _subs_key: (), subs_id: SeqID) {
func (r *registry[Payload]) Subscribe(_subsKey struct{}, subsID uint64) {
	// 	self.subscribers.insert(subs_id);
	r.subscribers[subsID] = struct{}{}
}

func (r *registry[Payload]) Unsubscribe(subsID uint64) {
	// 	self.subscribers.remove(&subs_id);
	delete(r.subscribers, subsID)
}

// impl<MakePayload, Payload, Error> Dispatch<MakePayload> for Registry
// where
// 	MakePayload: FnOnce() -> Result<Payload, Error>,
// 	Payload: Clone,
// {
// 	type Item = Payload;
// 	type Ret = Result<(), Error>;

// fn dispatch<F>(&mut self, make_payload: MakePayload, mut dispatch: F) -> Self::Ret
// where
//
//	F: FnMut(&SeqID, Self::Item),
//
// {
func (r *registry[Payload]) Dispatch(makePayload func() (Payload, error), dispatch func(uint64, Payload)) error {
	// 		let payload = make_payload()?;
	if len(r.subscribers) == 0 {
		return nil
	}
	payload, err := makePayload()
	if err != nil {
		return err
	}
	// 		if !self.subscribers.is_empty() {
	// 			let payload = make_payload()?;
	// 			for subs_id in &self.subscribers {
	// 				dispatch(subs_id, payload.clone());
	// 			}
	// 		}
	for subscriber := range r.subscribers {
		dispatch(subscriber, payload)
	}
	return nil
}
