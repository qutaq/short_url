package pool

import "sync"

// Resetter описывает типы, которые умеют сбрасывать своё состояние.
type Resetter interface {
	Reset()
}

// Pool — типизированная обёртка над sync.Pool для объектов с методом Reset.
type Pool[T Resetter] struct {
	pool sync.Pool
}

// New создаёт типизированный пул с обязательной фабрикой объектов.
// Передача nil в factory приводит к панике.
func New[T Resetter](factory func() T) *Pool[T] {
	if factory == nil {
		panic("pool: factory must not be nil")
	}
	p := &Pool[T]{}
	p.pool.New = func() any {
		return factory()
	}
	return p
}

// Get возвращает объект из пула.
func (p *Pool[T]) Get() T {
	v := p.pool.Get()
	if v == nil {
		var zero T
		return zero
	}
	return v.(T)
}

// Put сбрасывает состояние объекта и возвращает его обратно в пул.
func (p *Pool[T]) Put(v T) {
	v.Reset()
	p.pool.Put(v)
}
