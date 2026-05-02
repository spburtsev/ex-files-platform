package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache_NewInMemoryCache(t *testing.T) {
	c := NewInMemoryCache()
	c.Set("key", []byte("val"), 1*time.Minute)
	val, ok := c.Get("key")
	assert.True(t, ok)
	assert.Equal(t, []byte("val"), val)
}

func TestCache_SetAndGet(t *testing.T) {
	c := &InMemoryCache{entries: make(map[string]cacheEntry)}

	c.Set("key1", []byte("value1"), 1*time.Minute)
	val, ok := c.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, []byte("value1"), val)
}

func TestCache_GetMissing(t *testing.T) {
	c := &InMemoryCache{entries: make(map[string]cacheEntry)}

	val, ok := c.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestCache_Expiry(t *testing.T) {
	c := &InMemoryCache{entries: make(map[string]cacheEntry)}

	c.Set("key1", []byte("value1"), 1*time.Millisecond)
	time.Sleep(5 * time.Millisecond)

	val, ok := c.Get("key1")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestCache_Delete(t *testing.T) {
	c := &InMemoryCache{entries: make(map[string]cacheEntry)}

	c.Set("key1", []byte("value1"), 1*time.Minute)
	c.Delete("key1")

	val, ok := c.Get("key1")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestCache_Overwrite(t *testing.T) {
	c := &InMemoryCache{entries: make(map[string]cacheEntry)}

	c.Set("key1", []byte("value1"), 1*time.Minute)
	c.Set("key1", []byte("value2"), 1*time.Minute)

	val, ok := c.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, []byte("value2"), val)
}

func TestCache_MultipleKeys(t *testing.T) {
	c := &InMemoryCache{entries: make(map[string]cacheEntry)}

	c.Set("a", []byte("1"), 1*time.Minute)
	c.Set("b", []byte("2"), 1*time.Minute)
	c.Set("c", []byte("3"), 1*time.Minute)

	v1, ok1 := c.Get("a")
	v2, ok2 := c.Get("b")
	v3, ok3 := c.Get("c")

	assert.True(t, ok1)
	assert.Equal(t, []byte("1"), v1)
	assert.True(t, ok2)
	assert.Equal(t, []byte("2"), v2)
	assert.True(t, ok3)
	assert.Equal(t, []byte("3"), v3)
}

func TestCache_DeleteNonexistent(t *testing.T) {
	c := &InMemoryCache{entries: make(map[string]cacheEntry)}
	// Should not panic
	c.Delete("nonexistent")
}

func TestCache_JSONValues(t *testing.T) {
	c := &InMemoryCache{entries: make(map[string]cacheEntry)}

	jsonData := []byte(`{"id":1,"name":"Alice"}`)
	c.Set("user:1", jsonData, 1*time.Minute)

	val, ok := c.Get("user:1")
	assert.True(t, ok)
	assert.Equal(t, jsonData, val)
}
