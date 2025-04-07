package core

type CallContext uint8

const (
	CallContextOffchain CallContext = iota
	CallContextOnchain
)
