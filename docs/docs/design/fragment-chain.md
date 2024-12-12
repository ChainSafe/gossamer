# Fragment Chain 


### [Candidate Storage](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L189)

This structure does not care if candidates form a chain, it only stores candidates and holds "links" to a [`CandidateEntry`](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L346), those links are

`Parent Head Data Hash` to `Set<Candidate Hash>`
`Output Head Data Hash` to `Set<Candidate Hash>`

The parent head data hash is simply the parent block of that candidate, the output head data hash is the hash of the resultant [`HeadData`](https://github.com/paritytech/polkadot-sdk/blob/8f1606e9f9bd6269a4c2631a161dcc73e969a302/polkadot/parachain/src/primitives.rs#L51) after an execution, you can think of the output head data as the raw head of the parachain block that we get after validating the inputs against the parachain runtime.

`input -> execution -> head data (parachain block head)`, it is called head data because it is a `Vec<u8>`, what this head data contains is not for the validator, what we care is the *hash* of it.

### Scope

Scope is the context of a fragment chain, it contains constraints and ancestors that rules candidates, e.g. if a candidate was build in the context of an allowed relay parent.

### Fragment Node

This is a node that is part of a [`BackedChain`](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L596). When we create a new fragment we should make sure that the candidate is valid under the operating constraints. We use the constraints to build an expected persisted validation data and compare with the one we got from the candidate. A fragment node is always backed and it is generated from a [`CandidateEntry`](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L346).

### BackedChain

It holds a chain of backed nodes (fragments), in rust they store the fragments.

#### RevertToParentHash

This is a method where given a chain of fragments we need to remove all the fragments that descend a certain parent head data hash (this is the hash of the parachain block that is parent of another parachain block).

Let's say that we have the following [`BackedChain`](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L596)

```
{Parent Head Hash (0) | Output Head Data Hash (1)}
|
V
{Parent Head Hash (1) | Output Head Data Hash (2)}
|
V
{Parent Head Hash (2) | Output Head Data Hash (3)}
|
V
{Parent Head Hash (4) | Output Head Data Hash (5)}
|
V
{Parent Head Hash (5) | Output Head Data Hash (6)}
```

If we want to revert to `ParentHeadDataHash (3)` we search for a node that outputs such head data hash, in this case is the node at position 2, knowing the index we should remove its children (nodes at position 3 and 4) and clear their informations from the tables `byParentHead`, `byOutputHead` and `Candidates`, and we return the removed nodes.


### [Fragment Chain](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L663)

The fragment chain is a struct that an active leaf holds, also it should avoid cycles or paths in the three.

##### Populate Chain

This is a method from [`FragmentChain`](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L663) where given a scope and a Candidate Storage it will try to build a [`BackedChain`](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L596) with the candidates that follow the conditions:

- Does not introduce a cycle: That means, its output head data hash is not the parent head data hash of another candidate in the BackedChain.
- Its parent hash is correct: matches the previous output hash, forming a coherent chain
- Relay parent does not move backwards: given our earliest relay parent (the earliest ancestor that a candidate can use as its context) the current candidate's relay parent should not be before it.
- All non-pending-availability has a parent in the current scope.
- Candidates outputs fullfils the constraint.

If those conditions follows for 1 or more candidates we consider them possible backed/backable, however since the [`BackedChain`](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L596) does not allow forks we need to resolve that if there is more than one candidates, and the fork selection rule is: compare the hashes and the lowest hash will be chosen and pushed into the [`BackedChain`](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L596) and removed from the Candidate Storage


##### Trim Inelegible Forks

It does a depth breadth-first search looking for candidates in the Candidate Storage that have no potential and that are children of valid backed candidates.

Consider the following chain, the green letters forms the [`BackedChain`](https://github.com/paritytech/polkadot-sdk/blob/3d8da815ecd12b8f04daf87d6ffba5ec4a181806/polkadot/node/core/prospective-parachains/src/fragment_chain/mod.rs#L596) while the red letters are candidates placed in the `Unconnected` candidate storage, drawing them (without all the maps and hashes here and there) we can have something similar to the image below.

![IMG_944F8C1D62F0-1](./assets/img/parachain-protocol/pp_1.jpeg)

Now lets trim the inelegible forks out, but first we need to define the steps:

1. Lets build a queue of candidates in the backed chain (a.k.a green letters) and we will use an extra info called "has potential", which means that the candidate is good to stay connected (even if they form a fork).

2. queue is empty, the goto step 5, queue is not empty, pop the item from the start of the queue, mark it as visited and iterate over its children.

3. for each child:
    3.1. is the child in the unconnected hash set?
        - **true**:  go to next step
        - **false**: go to step 1
    3.2. did we already visited this child?
        - **true**: go to step 1
        - **false**: go to next step
    3.3. does the candidate and the child have potential?
        - **true**:  add to the queue as (child hash, true)
        - **false**: mark to remove, add to the queue as (child hash, false)

4. go to step 1

5. remove all items marked to be removed.

Lets apply the chain from image against the steps defined above, first things first, we should build the queue from our backed chain.

```
QUEUE     -> [ (A, true), (B, true), (C, true), (D, true) ]
VISITED   -> []
TO_REMOVE -> 
```

jumping to the next instruction, lets pop `(A, true)` from the queue, mark it as visited and iterate over its children!

```
QUEUE     -> [ (B, true), (C, true), (D, true) ]
VISITED   -> [ A ]
TO_REMOVE -> []
```

![IMG_88FB30D4B931-1](./assets/img/parachain-protocol/pp_2.jpeg)

them, child `B1` is in the unconnected set, is not visited, its parent (`A`) is potential, however `B1` is not, them we should mark it to remove and add to the queue as (`B1`, `false`) 

```
QUEUE     -> [ (B, true), (C, true), (D, true), (B1, false) ]
VISITED   -> [ A ]
TO_REMOVE -> [ B1 ]
```

the next child is `B`, which is not in the unconnected set, them since there is no children left we can just jump to the step 1.

continuing, lets pop `(B, true)` from the queue, mark it as visited and iterate over its children!

```
QUEUE     -> [ (C, true), (D, true), (B1, false) ]
VISITED   -> [ A, B ]
TO_REMOVE -> [ B1 ]
```

![IMG_319583A1BB18-1](./assets/img/parachain-protocol/pp_3.jpeg)

the node `B`, has children `C` and `C1`, starting with `C`, given `C` is in the `backed chain` it is not in the unconnected set, in this case we can go to the next child that is `C2`, that is in the unconnected set, was not visited, and has a parent (`B`) that is potential but `C2` itself is not, then lets push it to the queue as `(C2, false)` and mark it to remove.

```
QUEUE     -> [ (C, true), (D, true), (B1, false), (C2, false) ]
VISITED   -> [ A, B ]
TO_REMOVE -> [ B1, C2 ]
```

as there are no more children... lets pop `(C, true)` from the queue, mark it as visited and iterate over its children!

```
QUEUE     -> [ (D, true), (B1, false), (C2, false) ]
VISITED   -> [ A, B, C ]
TO_REMOVE -> [ B1, C2 ]
```

![IMG_C0062D30EB11-1](./assets/img/parachain-protocol/pp_4.jpeg)

the node `C` has only node `D` as its child, so since `D` is not in the unconnected set we can jump to step 1 again.

lets pop `(D, true)` from the queue, mark it as visited and iterate over its children...

```
QUEUE     -> [ (B1, false), (C2, false) ]
VISITED   -> [ A, B, C, D ]
TO_REMOVE -> [ B1, C2 ]
```

`D` has no children at all, them we can just stay in step 1 and pop `(B1, false)` from the queue, mark it as visited and iterate over its children.

```
QUEUE     -> [ (C2, false) ]
VISITED   -> [ A, B, C, D, B1 ]
TO_REMOVE -> [ B1, C2 ]
```

![IMG_86B6E79CE7E2-1](./assets/img/parachain-protocol/pp_5.jpeg)

The node `B1` has just one child that is `C1`, the child is present in the unconnected set, and is not visited but its parent `B1` is not a potential candidate, in this case we can directly mark `C1` having no potential as well and pushing it to the queue, and marking it to remove too.

```
QUEUE     -> [ (C2, false), (C1, false) ]
VISITED   -> [ A, B, C, D, B1 ]
TO_REMOVE -> [ B1, C2, C1 ]
```

going back to step 1 since there no more children left for `B1`, lets pop `(C2, false)` from the queue, mark it as visited and iterate over its children...

```
QUEUE     -> [ (C1, false) ]
VISITED   -> [ A, B, C, D, B1, C2 ]
TO_REMOVE -> [ B1, C2, C1 ]
```

![IMG_B913444820A7-1](./assets/img/parachain-protocol/pp_6.jpeg)

The node `C2` has just one child that is `D1`, the child is present in the unconnected set, and is not visited but its parent `C2` is not a potential candidate, in this case we can directly mark `D2` having no potential as well and pushing it to the queue, and marking it to remove too.

```
QUEUE     -> [ (C1, false), (D2, false) ]
VISITED   -> [ A, B, C, D, B1, C2 ]
TO_REMOVE -> [ B1, C2, C1, D2 ]
```

jumping back to step 1, just to be short we will pop `(C1, false)` and `(D2, false)` and as both entries don't have any children nothing will happen, so we drain the queue and since there is nothing more to process from the queue we can go to step 5.

```
QUEUE     -> [ ]
VISITED   -> [ A, B, C, D, B1, C2 ]
TO_REMOVE -> [ B1, C2, C1, D2 ]
```

step 5 says to remove every entry that is marked to remove from the unconnected set, that is a easy `delete(unconnected_set, entry)` operation, and we finish!

But I would like to highlight that, there is more complex there can exists and lead to a complete best chain reorg, for example:

- max_depth is 2 (a chain of max depth 3)
- `A -> B -> C` are the best backable chain.
- `D` is backed but would exceed the max depth.
- `F` is unconnected and seconded.
- `A1` has same parent as A, is backed but has a higher candidate hash. It'll therefore be deleted.
- `A1` has underneath a subtree that will all need to be trimmed. `A1 -> B1 -> C1` and `B1 -> C2`. (`C1` is backed).
- `A2` is seconded but is kept because it has a lower candidate hash than `A` (that is part of `checkPotential` method, where we use the fork selection rule to decide if a candidate has potential or not, even if the cadidate is not backed). `A2` points to `B2`, which is backed.
 
Check that D, F, A2 and B2 are kept as unconnected potential candidates, after trimming `A1` from the unconnected set.
