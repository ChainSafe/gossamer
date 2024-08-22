# trieDB


`triedb` is a library that acts as a layer over a key-value database, formatting those data into a radix-16 trie compatible with [polkadot spec](https://spec.polkadot.network/chap-state#sect-state-storage).  
It offers functionalities for writing and reading operations and uses lazy loading to fetch data from the database, as well as a caching system to optimize searches.

## Main features

- **Writes**: Basic functions to manipulate data in the trie.
- **Reads**: Basic functions to get data from the trie.
- **Lazy Loading**: Load data on demand.
- **Caching**: Enhances search performance.
- **Compatibility**: Works with any database implementing the `db.RWDatabase` interface and any cache implementing the `cache.TrieCache` interface.
- **Merkle proofs**: Create and verify merkle proofs.
- **Iterator**: Traverse the trie keys in order.

## Usage

### Create a Trie

To create an empty trie:

```go
trie := triedb.NewEmptyTrieDB(db)
```

### Insert Data

To insert a key and its associated value:

```go
err := trie.Put([]byte("key"), []byte("value"))
```

### Get Data

To get the value associated with a key:

```go
value := trie.Get([]byte("key"))
```

> Note: value will be empty `[]byte()` if there is no value for that key in the DB


### Remove Data

To delete a key and its associated value:

```go
trie.Delete([]byte("key"))
```

### Commit Data

All modifications happen in memory until a commit is applied to the database. The commit is automatically applied whenever the root hash of the trie is calculated using:

```go
rootHash, err := trie.Hash()
if err != nil {
    // handle error
}
```

or

```go
rootHash := trie.MustHash()
```

### Create a merkle proof

To create a merkle proof you will need a `db` previously created using a trie, the `trieVersion` (v0, v1), the `rootHash` for the trie in that db and an slice of `keys` for which the proof will be created.

```go
merkleProof, err := proof.NewMerkleProof(db, trieVersion, rootHash, keys) 
```

### Verify merkle proofs

In order to verify a merkle proof you can execute the `Verify` method over a previously created proof specifying the `trieVersion`, `rootHash` and the `items` you want to check, doing the following:

```go
err = merkleProof.Verify(trieVersion, rootHash, items)
if err != nil {
    fmt.Println("Invalid proof")
}
```

> Note: items is a slice of `proofItem` structure.   

### Iterator

There are two ways to use the key iterator.
Iterating by keys

```go
trieIterator := triedb.NewTrieDBIterator(trie)

for key := iter.NextKey(); key != nil; key = iter.NextKey() {
    fmt.Printf("key: %s", key)
}
```

Iterating by entries

```go
trieIterator := triedb.NewTrieDBIterator(trie)

for entry := iter.NextEntry(); entry != nil; entry = iter.NextEntry() {
    fmt.Printf("key: %s, value: %s", entry.Key, entry.Value)
}
```

