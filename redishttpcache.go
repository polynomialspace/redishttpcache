package redishttpcache

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type Config struct {
	Rdb          *redis.Client
	Expiration   time.Duration
	ErrCallback  func(error, *http.Request)
	HitCallback  func(*http.Request)
	MissCallback func(*http.Request)
	CacheRequest func(*http.Request) bool
	GenCacheKey  func(*http.Request) string
}

// TODO: context
func Middleware(next http.Handler, cfg *Config) http.Handler {
	if cfg.GenCacheKey == nil {
		cfg.GenCacheKey = func(r *http.Request) string {
			return "cache:" + r.URL.EscapedPath()
		}
	}
	if cfg.ErrCallback == nil {
		cfg.ErrCallback = func(_ error, _ *http.Request) {
			return
		}
	}
	if cfg.HitCallback == nil {
		cfg.HitCallback = func(_ *http.Request) {
			return
		}
	}
	if cfg.MissCallback == nil {
		cfg.MissCallback = func(_ *http.Request) {
			return
		}
	}
	if cfg.CacheRequest == nil {
		cfg.CacheRequest = func(r *http.Request) bool {
			return r.Method == "GET"
		}
	}

	ctx := context.Background()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cfg.CacheRequest(r) {
			next.ServeHTTP(w, r)
			return
		}

		cacheKey := cfg.GenCacheKey(r)
		content, err := cfg.Rdb.Get(ctx, cacheKey).Result()
		switch err {
		case redis.Nil:
			cfg.MissCallback(r)
			break
		case nil:
			cfg.HitCallback(r)
			var response Response
			err := json.Unmarshal([]byte(content), &response)
			if err != nil {
				cfg.ErrCallback(err, r)
				next.ServeHTTP(w, r)
				return
			}
			err = response.WriteHTTP(w)
			if err != nil {
				cfg.ErrCallback(err, r)
				return
			}
			return
		default:
			cfg.ErrCallback(err, r)
			next.ServeHTTP(w, r)
			return
		}

		//record and cache
		rec := httptest.NewRecorder()
		next.ServeHTTP(rec, r)
		result := rec.Result()
		if result.StatusCode > 500 {
			return
		}
		response := Response{
			Header: result.Header,
			Body:   rec.Body.Bytes(),
		}
		j, err := json.Marshal(response)
		err = cfg.Rdb.Set(ctx, cacheKey, j, cfg.Expiration).Err()
		if err != nil {
			cfg.ErrCallback(err, r)
			return
		}
		err = response.WriteHTTP(w)
		if err != nil {
			cfg.ErrCallback(err, r)
			return
		}
		return
	})
}

type Response struct {
	Header http.Header
	Body   []byte
}

func (cached Response) WriteHTTP(w http.ResponseWriter) error {
	for k, v := range cached.Header {
		w.Header().Set(k, strings.Join(v, ","))
	}
	_, err := w.Write(cached.Body)
	return err
}
