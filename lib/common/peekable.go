package common

import (
	"iter"
)

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
