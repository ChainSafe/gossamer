package triedb

import "errors"

var ErrInvalidStateRoot = errors.New("invalid state root")
var ErrIncompleteDB = errors.New("incomplete database")
var DecoderError = errors.New("corrupt trie item")
var InvalidHash = errors.New("hash is not value")
