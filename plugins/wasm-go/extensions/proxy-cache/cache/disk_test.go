package cache

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDiskCacheAllFunc(t *testing.T) {
	cache, err := NewDiskCache(DiskCacheOptions{
		RootDir:     "test/1.db",
		DiskLimit:   100,
		MemoryLimit: 100,
		TTL:         100,
	})
	require.NoError(t, err)
	require.NotNil(t, cache)
	err = cache.Set("test", []byte("test"))
	require.NoError(t, err)
	value, ok := cache.Get("test")
	require.Equal(t, true, ok)
	require.Equal(t, []byte("test"), value)
	err = cache.Delete("test")
	require.NoError(t, err)
}

func TestOutOfMemory(t *testing.T) {
	cache, err := NewDiskCache(DiskCacheOptions{
		RootDir:     "test/1.db",
		DiskLimit:   100,
		MemoryLimit: 1,
		TTL:         100,
	})
	require.NoError(t, err)
	require.NotNil(t, cache)
	err = cache.Set("test", []byte("test"))
	require.NoError(t, err)
	value, ok := cache.Get("test")
	require.Equal(t, true, ok)
	require.Equal(t, []byte("test"), value)
}
