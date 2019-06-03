package lockotron

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockedStruct struct {
	mock.Mock
}

func (ms *MockedStruct) Run() {
	ms.Called()
}

func TestCache_Set(t *testing.T) {
	config := NewConfig()
	config.DefaultTTL = 20 * time.Millisecond
	config.CleanupInterval = 1 * time.Millisecond

	t.Run("It expires the item using default ttl", func(t *testing.T) {
		cache := NewCache(config)
		cache.Set("key", "value")

		<-time.After(25 * time.Millisecond)
		value, err := cache.Get("key")

		require.Nil(t, value)
		require.Equal(t, ErrNotFound, err)
	})

	t.Run("It doesn't expire the item if default ttl hasn't been elapsed", func(t *testing.T) {
		cache := NewCache(config)
		cache.Set("key", "value")

		value, err := cache.Get("key")

		require.Equal(t, "value", value)
		require.Nil(t, err)
	})

	t.Run("It doesn't expire the item if ttl has been elapsed but cleaner is disabled", func(t *testing.T) {
		config.CleanupInterval = NoCleanup
		cache := NewCache(config)
		cache.Set("key", "value")

		<-time.After(25 * time.Millisecond)
		value, err := cache.Get("key")

		require.Equal(t, "value", value)
		require.Nil(t, err)
	})

	t.Run("It doesn't expire the item if no default ttl has been set", func(t *testing.T) {
		config.DefaultTTL = NoTTL
		cache := NewCache(config)
		cache.Set("key", "value")

		<-time.After(25 * time.Millisecond)
		value, err := cache.Get("key")

		require.Equal(t, "value", value)
		require.Nil(t, err)
	})

	t.Run("It stores the original object", func(t *testing.T) {
		cache := NewCache(config)

		type TestStruct struct {
			A int
		}
		original := TestStruct{A: 1}

		cache.Set("key", original)
		valueObj, err := cache.Get("key")
		value, ok := valueObj.(TestStruct)

		require.Equal(t, true, ok)
		require.Nil(t, err)
		require.Equal(t, original, value)
	})

	t.Run("It stores the pointer to original object", func(t *testing.T) {
		cache := NewCache(config)

		type TestStruct struct {
			A int
		}
		original := &TestStruct{A: 1}

		cache.Set("key", original)
		valueObj, err := cache.Get("key")
		value, ok := valueObj.(*TestStruct)

		require.Equal(t, true, ok)
		require.Nil(t, err)
		require.Equal(t, original, value)
	})
}

func TestCache_GetList(t *testing.T) {
	config := NewConfig()

	t.Run("It returns values for provided keys", func(t *testing.T) {
		cache := NewCache(config)
		cache.Set("key", "value")
		cache.Set("key1", "value1")
		cache.Set("key2", "value2")

		values1 := cache.GetList([]string{"key", "key2"})
		values2 := cache.GetList([]string{"key", "key1", "key2"})

		require.Equal(t, []interface{}{"value", "value2"}, values1)
		require.Equal(t, []interface{}{"value", "value1", "value2"}, values2)
	})

	t.Run("It skips missing items", func(t *testing.T) {
		cache := NewCache(config)
		cache.Set("key1", "value1")

		values := cache.GetList([]string{"key"})

		require.Equal(t, []interface{}{}, values)
	})
}

func TestCache_SetEx(t *testing.T) {
	config := NewConfig()
	config.CleanupInterval = 1 * time.Millisecond

	t.Run("It expires the item if custom ttl has been passed", func(t *testing.T) {
		cache := NewCache(config)
		cache.SetEx("key", 20*time.Millisecond, "value")

		<-time.After(25 * time.Millisecond)
		value, err := cache.Get("key")

		require.Nil(t, value)
		require.Equal(t, ErrNotFound, err)
	})

	t.Run("It doesn't expire the item if custom ttl has been passed but not elapsed", func(t *testing.T) {
		cache := NewCache(config)
		cache.SetEx("key", 20*time.Millisecond, "value")

		value, err := cache.Get("key")

		require.Equal(t, "value", value)
		require.Nil(t, err)
	})
}

func TestCache_Delete(t *testing.T) {
	config := NewConfig()

	t.Run("It deletes the value by key", func(t *testing.T) {
		cache := NewCache(config)
		cache.Set("key", "value")
		cache.Set("key1", "value1")

		cache.Delete("key")
		value, err := cache.Get("key")
		value1, err1 := cache.Get("key1")

		require.Nil(t, value)
		require.Equal(t, ErrNotFound, err)

		require.Nil(t, err1)
		require.Equal(t, "value1", value1)
	})
}

func TestCache_DeleteList(t *testing.T) {
	config := NewConfig()

	t.Run("It deletes values with given key list", func(t *testing.T) {
		cache := NewCache(config)
		cache.Set("key", "value")
		cache.Set("key1", "value1")
		cache.Set("key2", "value2")

		cache.DeleteList([]string{"key", "key2"})
		value, err := cache.Get("key")
		value1, err1 := cache.Get("key1")
		value2, err2 := cache.Get("key2")

		require.Nil(t, value)
		require.Equal(t, ErrNotFound, err)

		require.Nil(t, value2)
		require.Equal(t, ErrNotFound, err2)

		require.Nil(t, err1)
		require.Equal(t, "value1", value1)
	})
}

func TestCache_Fetch(t *testing.T) {
	config := NewConfig()
	config.DefaultTTL = 20 * time.Millisecond
	config.CleanupInterval = 1 * time.Millisecond

	t.Run("It calls fallback function and stores the return value if item is not set before", func(t *testing.T) {
		cache := NewCache(config)
		mocked := MockedStruct{}
		mocked.On("Run")

		value, err := cache.Fetch("key", func(string) (interface{}, error) {
			mocked.Run()

			return "value", nil
		})

		require.Nil(t, err)
		require.Equal(t, "value", value)
		mocked.AssertCalled(t, "Run")
	})

	t.Run("It doesn't call fallback function and returns the value if item is already set", func(t *testing.T) {
		cache := NewCache(config)
		mocked := MockedStruct{}
		mocked.On("Run")

		cache.Set("key", "value1")
		value, err := cache.Fetch("key", func(string) (interface{}, error) {
			mocked.Run()

			return "value", nil
		})

		require.Equal(t, "value1", value)
		require.Nil(t, err)
		mocked.AssertNotCalled(t, "Run")
	})

	t.Run("It returns the error produced by fallback function", func(t *testing.T) {
		cache := NewCache(config)
		expectedErr := errors.New("terrible error")

		value, err := cache.Fetch("key", func(string) (interface{}, error) {
			return nil, expectedErr
		})

		require.Equal(t, expectedErr, err)
		require.Nil(t, value)
	})

	t.Run("It expires values set by fallback function with default ttl", func(t *testing.T) {
		cache := NewCache(config)
		mocked := MockedStruct{}
		mocked.On("Run")

		value, err := cache.Fetch("key", func(string) (interface{}, error) {
			mocked.Run()

			return "value", nil
		})

		require.Nil(t, err)
		require.Equal(t, "value", value)
		mocked.AssertCalled(t, "Run")

		<-time.After(25 * time.Millisecond)

		value, err = cache.Get("key")
		require.Nil(t, value)
		require.Equal(t, ErrNotFound, err)
	})

	t.Run("It locks properly and calls fallback function only once on concurrent fetch", func(t *testing.T) {
		cache := NewCache(config)
		concurrency := 3
		var counter uint32

		for i := 0; i < concurrency; i++ {
			go cache.Fetch("key", func(string) (interface{}, error) {
				atomic.AddUint32(&counter, 1)

				time.Sleep(time.Millisecond)

				return "value", nil
			})
		}

		<-time.After(5 * time.Millisecond)

		counterFinal := atomic.LoadUint32(&counter)

		require.Equal(t, uint32(1), counterFinal)
	})
}

func TestCache_FetchEx(t *testing.T) {
	config := NewConfig()
	config.CleanupInterval = 1 * time.Millisecond

	t.Run("It expires values set by fallback function with custom ttl", func(t *testing.T) {
		cache := NewCache(config)
		mocked := MockedStruct{}
		mocked.On("Run")

		value, err := cache.FetchEx("key", 20*time.Millisecond, func(string) (interface{}, error) {
			mocked.Run()

			return "value", nil
		})

		require.Nil(t, err)
		require.Equal(t, "value", value)
		mocked.AssertCalled(t, "Run")

		<-time.After(25 * time.Millisecond)

		value, err = cache.Get("key")
		require.Nil(t, value)
		require.Equal(t, ErrNotFound, err)
	})
}
