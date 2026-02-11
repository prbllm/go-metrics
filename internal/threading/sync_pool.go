package threading

import (
	"reflect"
	"sync"
)

// Pool определяет безопасную обёртку над sync.Pool
type Pool[T Resetter] struct {
	storage sync.Pool
	new     func() T
}

// New создаёт пул с фабрикой f для создания значений типа T.
func New[T Resetter](f func() T) *Pool[T] {
	return &Pool[T]{
		storage: sync.Pool{
			New: func() any { return f() },
		},
		new: f,
	}
}

// Get возвращает значение из пула (или новое, если пул пуст).
func (p *Pool[T]) Get() T {
	v := p.storage.Get()
	if v == nil {
		return p.new()
	}
	return v.(T)
}

// Put возвращает значение в пул, сбрасывая значения структуры через Reset.
func (p *Pool[T]) Put(v T) {
	if isNil(v) {
		return
	}
	v.Reset()
	p.storage.Put(v)
}

func isNil[T any](v T) bool {
	if any(v) == nil {
		return true
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Interface:
		if rv.IsNil() {
			return true
		}
		return isNil(rv.Elem().Interface())
	case reflect.Pointer, reflect.Chan, reflect.Func, reflect.Map, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}
