// Copyright 2024 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package trie

import (
	"testing"

	hashdb "github.com/ChainSafe/gossamer/internal/hash-db"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hash"
	"github.com/ChainSafe/gossamer/internal/primitives/core/hasher"
	"github.com/ChainSafe/gossamer/pkg/trie/triedb"
	"github.com/stretchr/testify/require"
)

var (
	_ hashdb.HashDB[hash.H256] = &KeyspacedDB[hash.H256]{}
)

// fn check_equivalent<T: TrieConfiguration>(input: &Vec<(&[u8], &[u8])>) {
func checkEquivalent(t *testing.T, input []KeyValue, layout Layout[hash.H256]) {

	//	{
	//		let closed_form = T::trie_root(input.clone());
	//		let d = T::trie_root_unhashed(input.clone());
	//		println!("Data: {:#x?}, {:#x?}", d, Blake2Hasher::hash(&d[..]));
	//		let persistent = {
	//			let mut memdb = MemoryDBMeta::default();
	//			let mut root = Default::default();
	//			let mut t = TrieDBMutBuilder::<T>::new(&mut memdb, &mut root).build();
	//			for (x, y) in input.iter().rev() {
	//				t.insert(x, y).unwrap();
	//			}
	//			*t.root()
	//		};
	//		assert_eq!(closed_form, persistent);
	//	}
	closedForm := layout.TrieRoot(input)

	t.Logf("%s", closedForm)

	memDB := NewMemoryDB[hash.H256, hasher.Blake2Hasher]()
	trieDB := triedb.NewEmptyTrieDB[hash.H256, hasher.Blake2Hasher](memDB, layout)
	for i := len(input) - 1; i >= 0; i-- {
		kv := input[i]
		trieDB.Set(kv.Key, kv.Value)
	}
	persistent := trieDB.MustHash()
	require.Equal(t, closedForm, persistent)
}

// fn check_iteration<T: TrieConfiguration>(input: &Vec<(&[u8], &[u8])>) {
// 	let mut memdb = MemoryDBMeta::default();
// 	let mut root = Default::default();
// 	{
// 		let mut t = TrieDBMutBuilder::<T>::new(&mut memdb, &mut root).build();
// 		for (x, y) in input.clone() {
// 			t.insert(x, y).unwrap();
// 		}
// 	}
// 	{
// 		let t = TrieDBBuilder::<T>::new(&memdb, &root).build();
// 		assert_eq!(
// 			input.iter().map(|(i, j)| (i.to_vec(), j.to_vec())).collect::<Vec<_>>(),
// 			t.iter()
// 				.unwrap()
// 				.map(|x| x.map(|y| (y.0, y.1.to_vec())).unwrap())
// 				.collect::<Vec<_>>()
// 		);
// 	}
// }

//	fn check_input(input: &Vec<(&[u8], &[u8])>) {
//		check_equivalent::<LayoutV0>(input);
//		check_iteration::<LayoutV0>(input);
//		check_equivalent::<LayoutV1>(input);
//		check_iteration::<LayoutV1>(input);
//	}
func checkInput(t *testing.T, input []KeyValue) {
	checkEquivalent(t, input, LayoutV0[hasher.Blake2Hasher, hash.H256]{})
	// check_iteration::<LayoutV0>(input)
	checkEquivalent(t, input, LayoutV1[hasher.Blake2Hasher, hash.H256]{})
	// check_iteration::<LayoutV1>(input)
}

func TestTrieRoot(t *testing.T) {
	// fn default_trie_root() {
	t.Run("default_trie_root", func(t *testing.T) {
		// 	let mut db = MemoryDB::default();
		db := NewMemoryDB[hash.H256, hasher.Blake2Hasher]()
		// 	let mut root = TrieHash::<LayoutV1>::default();
		// root := hash.NewH256()
		// 	let mut empty = TrieDBMutBuilder::<LayoutV1>::new(&mut db, &mut root).build();
		empty := triedb.NewEmptyTrieDB[hash.H256, hasher.Blake2Hasher](db, LayoutV1[hasher.Blake2Hasher, hash.H256]{})
		// 	empty.commit();
		root1 := empty.MustHash()
		// 	let root1 = empty.root().as_ref().to_vec();
		// 	let root2: Vec<u8> = LayoutV1::trie_root::<_, Vec<u8>, Vec<u8>>(std::iter::empty())
		// 		.as_ref()
		// 		.iter()
		// 		.cloned()
		// 		.collect();
		root2 := LayoutV1[hasher.Blake2Hasher, hash.H256]{}.TrieRoot(nil)
		require.Equal(t, root1, root2)
		// 	assert_eq!(root1, root2);
	})

	// fn empty_is_equivalent() {
	t.Run("empty_is_equivalent", func(t *testing.T) {
		// 	let input: Vec<(&[u8], &[u8])> = vec![];
		// 	check_input(&input);
		checkInput(t, nil)
	})

	// fn leaf_is_equivalent() {
	t.Run("leaf_is_equivalent", func(t *testing.T) {
		// let input: Vec<(&[u8], &[u8])> = vec![(&[0xaa][..], &[0xbb][..])];
		// check_input(&input);
		checkInput(t, []KeyValue{
			{Key: []byte{0xaa}, Value: []byte{0xbb}},
		})
	})

	// #[test]
	// fn branch_is_equivalent() {
	t.Run("branch_is_equivalent", func(t *testing.T) {
		// 	let input: Vec<(&[u8], &[u8])> =
		// 		vec![(&[0xaa][..], &[0x10][..]), (&[0xba][..], &[0x11][..])];
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte{0x10}},
			{Key: []byte{0xba}, Value: []byte{0x11}},
		}
		// 	check_input(&input);
		checkInput(t, input)
	})

	// #[test]
	// fn extension_and_branch_is_equivalent() {
	t.Run("extension_and_branch_is_equivalent", func(t *testing.T) {
		// 	let input: Vec<(&[u8], &[u8])> =
		// 		vec![(&[0xaa][..], &[0x10][..]), (&[0xab][..], &[0x11][..])];
		// 	check_input(&input);
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte{0x10}},
			{Key: []byte{0xab}, Value: []byte{0x11}},
		}
		checkInput(t, input)
	})

	// #[test]
	// fn standard_is_equivalent() {
	// 	let st = StandardMap {
	// 		alphabet: Alphabet::All,
	// 		min_key: 32,
	// 		journal_key: 0,
	// 		value_mode: ValueMode::Random,
	// 		count: 1000,
	// 	};
	// 	let mut d = st.make();
	// 	d.sort_by(|(a, _), (b, _)| a.cmp(b));
	// 	let dr = d.iter().map(|v| (&v.0[..], &v.1[..])).collect();
	// 	check_input(&dr);
	// }

	// #[test]
	// fn extension_and_branch_with_value_is_equivalent() {
	t.Run("extension_and_branch_with_value_is_equivalent", func(t *testing.T) {
		// 	let input: Vec<(&[u8], &[u8])> = vec![
		// 		(&[0xaa][..], &[0xa0][..]),
		// 		(&[0xaa, 0xaa][..], &[0xaa][..]),
		// 		(&[0xaa, 0xbb][..], &[0xab][..]),
		// 	];
		// 	check_input(&input);
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte{0xa0}},
			{Key: []byte{0xaa, 0xaa}, Value: []byte{0xaa}},
			{Key: []byte{0xaa, 0xbb}, Value: []byte{0xab}},
		}
		checkInput(t, input)
	})

	// #[test]
	// fn bigger_extension_and_branch_with_value_is_equivalent() {
	t.Run("bigger_extension_and_branch_with_value_is_equivalent", func(t *testing.T) {
		// 	let input: Vec<(&[u8], &[u8])> = vec![
		// 		(&[0xaa][..], &[0xa0][..]),
		// 		(&[0xaa, 0xaa][..], &[0xaa][..]),
		// 		(&[0xaa, 0xbb][..], &[0xab][..]),
		// 		(&[0xbb][..], &[0xb0][..]),
		// 		(&[0xbb, 0xbb][..], &[0xbb][..]),
		// 		(&[0xbb, 0xcc][..], &[0xbc][..]),
		// 	];
		// 	check_input(&input);
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte{0xa0}},
			{Key: []byte{0xaa, 0xaa}, Value: []byte{0xaa}},
			{Key: []byte{0xaa, 0xbb}, Value: []byte{0xab}},
			{Key: []byte{0xbb}, Value: []byte{0xb0}},
			{Key: []byte{0xbb, 0xbb}, Value: []byte{0xbb}},
			{Key: []byte{0xbb, 0xcc}, Value: []byte{0xbc}},
		}
		checkInput(t, input)
	})

	// #[test]
	// fn single_long_leaf_is_equivalent() {
	t.Run("single_long_leaf_is_equivalent", func(t *testing.T) {
		// 	let input: Vec<(&[u8], &[u8])> = vec![
		// 		(
		// 			&[0xaa][..],
		// 			&b"ABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABC"[..],
		// 		),
		// 		(&[0xba][..], &[0x11][..]),
		// 	];
		// 	check_input(&input);
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte("ABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABC")},
			{Key: []byte{0xba}, Value: []byte{0x11}},
		}
		checkInput(t, input)
	})

	// #[test]
	// fn two_long_leaves_is_equivalent() {
	t.Run("two_long_leaves_is_equivalent", func(t *testing.T) {
		// 	let input: Vec<(&[u8], &[u8])> = vec![
		// 		(
		// 			&[0xaa][..],
		// 			&b"ABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABC"[..],
		// 		),
		// 		(
		// 			&[0xba][..],
		// 			&b"ABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABC"[..],
		// 		),
		// 	];
		// 	check_input(&input);
		input := []KeyValue{
			{Key: []byte{0xaa}, Value: []byte("ABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABC")},
			{Key: []byte{0xba}, Value: []byte("ABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABCABC")},
		}
		checkInput(t, input)
	})

}
