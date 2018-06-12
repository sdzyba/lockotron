package imp

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type CacheTestSuite struct {
	suite.Suite
}

func TestCacheTestSuite(t *testing.T) {
	suite.Run(t, &CacheTestSuite{})
}

func (suite *CacheTestSuite) Case(name string, testFn func(*testing.T)) {
	suite.T().Run(name, testFn)
}

type MockedStruct struct {
	mock.Mock
}

func (ms *MockedStruct) Run() {
	ms.Called()
}

func (suite *CacheTestSuite) TestSet() {
	config := NewConfig()
	config.DefaultTTL = 20 * time.Millisecond
	config.CleanupInterval = 1 * time.Millisecond

	suite.Case("It expires the item using default ttl", func(t *testing.T) {
		cache := NewCache(config)
		cache.Set("key", "value")

		<-time.After(25 * time.Millisecond)
		value, err := cache.Get("key")

		assert.Nil(t, value)
		assert.Equal(t, ErrNotFound, err)
	})

	suite.Case("It doesn't expire the item if default ttl hasn't been elapsed", func(t *testing.T) {
		cache := NewCache(config)
		cache.Set("key", "value")

		value, err := cache.Get("key")

		assert.Equal(t, "value", value)
		assert.Nil(t, err)
	})

	suite.Case("It doesn't expire the item if ttl has been elapsed but cleaner is disabled", func(t *testing.T) {
		config.CleanupInterval = NoCleaner
		cache := NewCache(config)
		cache.Set("key", "value")

		<-time.After(25 * time.Millisecond)
		value, err := cache.Get("key")

		assert.Equal(t, "value", value)
		assert.Nil(t, err)
	})

	suite.Case("It doesn't expire the item if no default ttl has been set", func(t *testing.T) {
		config.DefaultTTL = NoTTL
		cache := NewCache(config)
		cache.Set("key", "value")

		<-time.After(25 * time.Millisecond)
		value, err := cache.Get("key")

		assert.Equal(t, "value", value)
		assert.Nil(t, err)
	})

	suite.Case("It stores the original object", func(t *testing.T) {
		cache := NewCache(config)

		type TestStruct struct {
			A int
		}
		original := TestStruct{A: 1}

		cache.Set("key", original)
		valueObj, err := cache.Get("key")
		value, ok := valueObj.(TestStruct)

		assert.Equal(t, true, ok)
		assert.Nil(t, err)
		assert.Equal(t, original, value)
	})

	suite.Case("It stores the pointer to original object", func(t *testing.T) {
		cache := NewCache(config)

		type TestStruct struct {
			A int
		}
		original := &TestStruct{A: 1}

		cache.Set("key", original)
		valueObj, err := cache.Get("key")
		value, ok := valueObj.(*TestStruct)

		assert.Equal(t, true, ok)
		assert.Nil(t, err)
		assert.Equal(t, original, value)
	})
}

func (suite *CacheTestSuite) TestSetEx() {
	config := NewConfig()
	config.CleanupInterval = 1 * time.Millisecond

	suite.Case("It expires the item if custom ttl has been passed", func(t *testing.T) {
		cache := NewCache(config)
		cache.SetEx("key", 20*time.Millisecond, "value")

		<-time.After(25 * time.Millisecond)
		value, err := cache.Get("key")

		assert.Nil(t, value)
		assert.Equal(t, ErrNotFound, err)
	})

	suite.Case("It doesn't expire the item if custom ttl has been passed but not elapsed", func(t *testing.T) {
		cache := NewCache(config)
		cache.SetEx("key", 20*time.Millisecond, "value")

		value, err := cache.Get("key")

		assert.Equal(t, "value", value)
		assert.Nil(t, err)
	})
}

func (suite *CacheTestSuite) TestDelete() {
	config := NewConfig()

	suite.Case("It deletes the value by key", func(t *testing.T) {
		cache := NewCache(config)
		cache.Set("key", "value")
		cache.Set("key1", "value1")

		cache.Delete("key")
		value, err := cache.Get("key")
		value1, err1 := cache.Get("key1")

		assert.Nil(t, value)
		assert.Equal(t, ErrNotFound, err)

		assert.Nil(t, err1)
		assert.Equal(t, "value1", value1)
	})
}

func (suite *CacheTestSuite) TestFetch() {
	config := NewConfig()
	config.DefaultTTL = 20 * time.Millisecond
	config.CleanupInterval = 1 * time.Millisecond

	suite.Case("It calls fallback function and stores the return value if item is not set before", func(t *testing.T) {
		cache := NewCache(config)
		mocked := MockedStruct{}
		mocked.On("Run")

		value, err := cache.Fetch("key", func(string) (interface{}, error) {
			mocked.Run()

			return "value", nil
		})

		assert.Nil(t, err)
		assert.Equal(t, "value", value)
		mocked.AssertCalled(t, "Run")
	})

	suite.Case("It doesn't call fallback function and returns the value if item is already set", func(t *testing.T) {
		cache := NewCache(config)
		mocked := MockedStruct{}
		mocked.On("Run")

		cache.Set("key", "value1")
		value, err := cache.Fetch("key", func(string) (interface{}, error) {
			mocked.Run()

			return "value", nil
		})

		assert.Equal(t, "value1", value)
		assert.Nil(t, err)
		mocked.AssertNotCalled(t, "Run")
	})

	suite.Case("It returns the error produced by fallback function", func(t *testing.T) {
		cache := NewCache(config)
		expectedErr := errors.New("terrible error")

		value, err := cache.Fetch("key", func(string) (interface{}, error) {
			return nil, expectedErr
		})

		assert.Equal(t, expectedErr, err)
		assert.Nil(t, value)
	})

	suite.Case("It expires values set by fallback function with default ttl", func(t *testing.T) {
		cache := NewCache(config)
		mocked := MockedStruct{}
		mocked.On("Run")

		value, err := cache.Fetch("key", func(string) (interface{}, error) {
			mocked.Run()

			return "value", nil
		})

		assert.Nil(t, err)
		assert.Equal(t, "value", value)
		mocked.AssertCalled(t, "Run")

		<-time.After(25 * time.Millisecond)

		value, err = cache.Get("key")
		assert.Nil(t, value)
		assert.Equal(t, ErrNotFound, err)
	})
}

func (suite *CacheTestSuite) TestFetchEx() {
	config := NewConfig()
	config.CleanupInterval = 1 * time.Millisecond

	suite.Case("It expires values set by fallback function with custom ttl", func(t *testing.T) {
		cache := NewCache(config)
		mocked := MockedStruct{}
		mocked.On("Run")

		value, err := cache.FetchEx("key", 20*time.Millisecond, func(string) (interface{}, error) {
			mocked.Run()

			return "value", nil
		})

		assert.Nil(t, err)
		assert.Equal(t, "value", value)
		mocked.AssertCalled(t, "Run")

		<-time.After(25 * time.Millisecond)

		value, err = cache.Get("key")
		assert.Nil(t, value)
		assert.Equal(t, ErrNotFound, err)
	})
}
