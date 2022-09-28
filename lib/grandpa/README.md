## Politeness in Grandpa

This document defines the rules that make up an implementation of polite grandpa. These rules include 
what actions are reported, the cost/benefit of the action, and what events trigger these actions.
Each section will represent an action and define all relevant information about that action.

## Relevant Links

The costs/benefits in substrate are defined [here](https://github.com/paritytech/substrate/blob/88731ed619f6be1280f01cc8f02f10e2fb4cf199/client/finality-grandpa/src/communication/mod.rs#L93)

The misbehavior function that calculates cost is defined [here](https://github.com/paritytech/substrate/blob/88731ed619f6be1280f01cc8f02f10e2fb4cf199/client/finality-grandpa/src/communication/gossip.rs#L434)

## Costs

Init with one example, will be added to as politness is implemented

### PAST_REJECTION

**Description** TODO insert description

**Cost** -50

**Message** "Grandpa: Past message"

**Occurences** TODO insert list of events that trigger this action and links to events in gossamer code

## Benefits

TODO