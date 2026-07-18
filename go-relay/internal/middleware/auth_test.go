package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ThreeHats/foundryvtt-rest-api-relay/go-relay/internal/model"
)

func TestKeyPrefix(t *testing.T) {
	if got := keyPrefix("short"); got != "short" {
		t.Errorf("expected short key returned as-is, got %q", got)
	}
	if got := keyPrefix("0123456789abcdef"); got != "01234567" {
		t.Errorf("expected 8-char prefix, got %q", got)
	}
}

func TestHasBearerHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if hasBearerHeader(r) {
		t.Error("expected no bearer header on bare request")
	}

	r.Header.Set("Authorization", "Bearer abc123")
	if !hasBearerHeader(r) {
		t.Error("expected bearer header to be detected")
	}

	r.Header.Set("Authorization", "Basic abc123")
	if hasBearerHeader(r) {
		t.Error("expected Basic auth to not be treated as Bearer")
	}
}

func TestAuthCacheSetGetInvalidate(t *testing.T) {
	// Isolate from other tests / the background cleanup goroutine.
	authCacheMu.Lock()
	authCacheMap = make(map[string]*authCacheEntry)
	authCacheMu.Unlock()

	user := &model.User{ID: 42}
	setCachedAuth("key-1", user, nil)

	entry, ok := getCachedAuth("key-1")
	if !ok || entry.user.ID != 42 {
		t.Fatalf("expected cached entry for key-1, got ok=%v entry=%v", ok, entry)
	}

	InvalidateCachedAuth("key-1")
	if _, ok := getCachedAuth("key-1"); ok {
		t.Error("expected cache entry to be gone after invalidation")
	}
}

func TestAuthCacheExpiry(t *testing.T) {
	authCacheMu.Lock()
	authCacheMap = make(map[string]*authCacheEntry)
	authCacheMap["expired"] = &authCacheEntry{
		user:      &model.User{ID: 1},
		expiresAt: time.Now().Add(-time.Second),
	}
	authCacheMu.Unlock()

	if _, ok := getCachedAuth("expired"); ok {
		t.Error("expected expired entry to be treated as a cache miss")
	}
}

func TestPruneCacheRemovesOnlyExpiredEntries(t *testing.T) {
	authCacheMu.Lock()
	authCacheMap = map[string]*authCacheEntry{
		"fresh":   {user: &model.User{ID: 1}, expiresAt: time.Now().Add(time.Minute)},
		"expired": {user: &model.User{ID: 2}, expiresAt: time.Now().Add(-time.Minute)},
	}
	authCacheMu.Unlock()

	pruneCache()

	authCacheMu.RLock()
	_, freshOK := authCacheMap["fresh"]
	_, expiredOK := authCacheMap["expired"]
	authCacheMu.RUnlock()

	if !freshOK {
		t.Error("expected fresh entry to survive pruning")
	}
	if expiredOK {
		t.Error("expected expired entry to be pruned")
	}
}

func TestInvalidateCachedAuthForUser(t *testing.T) {
	authCacheMu.Lock()
	authCacheMap = map[string]*authCacheEntry{
		"key-a": {user: &model.User{ID: 1}, expiresAt: time.Now().Add(time.Minute)},
		"key-b": {user: &model.User{ID: 2}, expiresAt: time.Now().Add(time.Minute)},
	}
	authCacheMu.Unlock()

	InvalidateCachedAuthForUser(1)

	if _, ok := getCachedAuth("key-a"); ok {
		t.Error("expected key-a to be invalidated for user 1")
	}
	if _, ok := getCachedAuth("key-b"); !ok {
		t.Error("expected key-b (user 2) to remain cached")
	}
}
