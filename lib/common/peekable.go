package common

import (
	"iter"
)

// Peekable wraps a Seq[V] so you can look at the next element
// without consuming it.
type Peekable[V any] struct {
	next      func() (V, bool)
	stop      func()
	buffer    V
	hasBuffer bool
}

// NewPeekable turns any iter.Seq[V] into a *Peekable[V].
// It calls iter.Pull to get the pull‐style next/stop functions. :contentReference[oaicite:0]{index=0}
func NewPeekable[V any](seq iter.Seq[V]) *Peekable[V] {
	next, stop := iter.Pull(seq)
	return &Peekable[V]{next: next, stop: stop}
}

// Peek returns (value, true) for the next element without consuming it.
// If the sequence is exhausted, it returns (zero, false).
func (p *Peekable[V]) Peek() (V, bool) {
	if p.hasBuffer {
		return p.buffer, true
	}
	v, ok := p.next()
	if !ok {
		var zero V
		return zero, false
	}
	p.buffer = v
	p.hasBuffer = true
	return v, true
}

// Next returns (value, true) for the next element, consuming it.
// If you've already Peeked, it returns the buffered value.
func (p *Peekable[V]) Next() (V, bool) {
	if p.hasBuffer {
		v := p.buffer
		p.hasBuffer = false
		return v, true
	}
	return p.next()
}

// Close must be called when you're done early to free any resources.
func (p *Peekable[V]) Close() {
	p.stop()
}

// Peekable2 wraps a Seq2[K, V] so you can mirar (peek) la siguiente
// pareja sin consumirla.
type Peekable2[K, V any] struct {
	next      func() (K, V, bool)
	stop      func()
	bufK      K
	bufV      V
	hasBuffer bool
}

// NewPeekable2 convierte un iter.Seq2[K, V] en *Peekable2[K, V].
// Internamente usa iter.Pull2. :contentReference[oaicite:0]{index=0}
func NewPeekable2[K, V any](seq iter.Seq2[K, V]) *Peekable2[K, V] {
	next, stop := iter.Pull2(seq)
	return &Peekable2[K, V]{next: next, stop: stop}
}

// Peek devuelve (k, v, true) de la siguiente pareja sin consumirla.
// Si la secuencia ya terminó, retorna (zero, zero, false).
func (p *Peekable2[K, V]) Peek() (K, V, bool) {
	if p.hasBuffer {
		return p.bufK, p.bufV, true
	}
	k, v, ok := p.next()
	if !ok {
		var zk K
		var zv V
		return zk, zv, false
	}
	p.bufK, p.bufV, p.hasBuffer = k, v, true
	return k, v, true
}

// Next devuelve (k, v, true) consumiendo la siguiente pareja.
// Si ya se hizo Peek, devuelve lo que esté en el buffer.
func (p *Peekable2[K, V]) Next() (K, V, bool) {
	if p.hasBuffer {
		p.hasBuffer = false
		return p.bufK, p.bufV, true
	}
	return p.next()
}

// Close debe llamarse (normalmente con defer) si sales antes de
// consumir todo, para permitir que el iterador “push” termine.
// Equivale a llamar a stop() de iter.Pull2. :contentReference[oaicite:1]{index=1}
func (p *Peekable2[K, V]) Close() {
	p.stop()
}
