package threading

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

type countingResetter struct {
	resetCount int
}

func (c *countingResetter) Reset() {
	c.resetCount++
}

func TestNew_ReturnsNonNil(t *testing.T) {
	pool := New(func() *countingResetter { return &countingResetter{} })
	require.NotNil(t, pool, "New must return non-nil pool")
}

func TestPool_Get_WithoutPut_ReturnsValueFromFactory(t *testing.T) {
	callCount := 0
	pool := New(func() *countingResetter {
		callCount++
		return &countingResetter{}
	})

	v := pool.Get()
	require.NotNil(t, v, "Get must return non-nil value")
	require.Equal(t, 1, callCount, "Get on empty pool must use factory once")
	require.Equal(t, 0, v.resetCount, "New value must not have Reset called")
}

func TestPool_Put_CallsReset(t *testing.T) {
	pool := New(func() *countingResetter { return &countingResetter{} })
	v := pool.Get()
	require.Equal(t, 0, v.resetCount)

	pool.Put(v)
	require.Equal(t, 1, v.resetCount, "Put must call Reset before storing")
}

func TestPool_PutTypedNil_DoesNotPanic(t *testing.T) {
	pool := New(func() *countingResetter { return &countingResetter{} })
	require.NotPanics(t, func() {
		pool.Put((*countingResetter)(nil))
	}, "Put(typed nil) must not panic")
}

func TestPool_GetAfterPut_ReturnsResetObject(t *testing.T) {
	pool := New(func() *countingResetter { return &countingResetter{} })

	v := pool.Get()
	require.Equal(t, 0, v.resetCount, "new value: Reset not called yet")
	pool.Put(v)

	got := pool.Get()
	require.NotNil(t, got)
	if got == v {
		require.Equal(t, 1, got.resetCount, "reused value must have been Reset once in Put")
	}
}

func TestPool_ConcurrentGetPut(t *testing.T) {
	pool := New(func() *countingResetter { return &countingResetter{} })

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				v := pool.Get()
				require.NotNil(t, v)
				pool.Put(v)
			}
		}()
	}
	wg.Wait()
}
