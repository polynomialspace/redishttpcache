a redis http cache in go.

see `example/example.go` for basic usage.

various options can be set via the `Config` struct;
```
type Config struct {
	Rdb          *redis.Client
	Expiration   time.Duration
	ErrCallback  func(error, *http.Request)
	HitCallback  func(*http.Request)
	MissCallback func(*http.Request)
	CacheRequest func(*http.Request) bool
	GenCacheKey  func(*http.Request) string
}
```

`Rdb` and `Expiration` are both handled by redis itself, being the redis client config and the key expiration for all keys set by `Middleware`. redis uses a value of `-1` to indicate no key expiration.

the `Err`, `Hit`, and `Miss` callbacks are all functions called during the `Middleware` logic, corresponding to caught errors, cache hits, and cache misses respectively.
The `next` handler will always be called in the case of an error or miss.

`CacheRequest` can be used to configure when a request is cached. By default it is configured to only cache `GET` requests.

`GenCacheKey` can be used to generate the redis key name for each request. By default it returns `"cache:" + r.URL.EscapedPath()`