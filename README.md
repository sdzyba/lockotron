# Lockotron

In-memory cache with thread-safe fetch mechanism

## Example

```go
config := lockotron.NewConfig()

// Default cached item TTL.
// Set to `cache.NoTTL` is you want to store your value in cache without expiration.
// Defaults to 15 minutes
config.DefaultTTL = 1 * time.Hour

// Specify how often cache should be cleaned up.
// Set to `cache.NoCleanup` if you don't want your cache items to be cleaned up automatically.
config.CleanupInterval = 30 * time.Minute // default is 20 minutes

cache := lockotron.NewCache(config)

// Set an item with key "foo" and value "bar"
cache.Set("foo", "bar")

var (
	err error
	value interface{}
)

value, err = cache.Get("foo")
fmt.Println(err)
// Output: <nil>
fmt.Println(value.(string))
// Output: "bar"

value, err = cache.Get("baz")
fmt.Println(err)
// Output: cached value not found
fmt.Println(lockotron.ErrNotFound == err)
// Output: true
fmt.Println(value)
// Output: <nil>

// Fetch looks for an existing item with given key at first and returns it if found.
// If item isn't set, it calls provided fallback function, returned result of which is stored in a cache.
value, err = cache.Fetch("foo", func(string) (interface{}, error) {
	return "bar2", nil
})
// Return value is still "bar", since "foo" was already set before
fmt.Println(value.(string))
// Output: "bar"

value, err = cache.Fetch("newfoo", func(string) (interface{}, error) {
	return "newbar", nil
})
fmt.Println(value.(string))
// Output: "newbar"
```

## Fetch
Concurrent calls of `Fetch()` are completely thread safe, meaning the fallback function will be called only once despite how many goroutines will try to fetch the value.

First goroutine locks the access by the given key and proceeds dealing with the fallback function. All awaiting goroutines will receive value that has been set once first goroutine unlocks the access.

If value was already set, `Fetch()` behaves the same as `Get()`.

## More info
Please refer to [tests](https://github.com/sdzyba/lockotron/blob/master/cache_test.go) to see more use cases.
