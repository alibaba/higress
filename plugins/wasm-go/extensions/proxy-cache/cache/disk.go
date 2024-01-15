package cache

import (
	"github.com/ByteStorage/FlyDB/config"
	"github.com/ByteStorage/FlyDB/engine"
	"github.com/ByteStorage/FlyDB/flydb"
)

type diskCache struct {
	db *engine.DB
}

func NewDiskCache(rootDir string, limit int) (Cache, error) {
	options := config.DefaultOptions
	options.DirPath = rootDir
	db, err := flydb.NewFlyDB(options)
	if err != nil {
		return nil, err
	}
	return &diskCache{
		db: db,
	}, nil
}

func (c *diskCache) Get(key string) ([]byte, bool) {
	bytes, err := c.db.Get([]byte(key))
	if err != nil {
		return nil, false
	}
	return bytes, true
}

func (c *diskCache) Set(key string, value []byte) error {
	return c.db.Put([]byte(key), value)
}

func (c *diskCache) Delete(key string) error {
	return c.db.Delete([]byte(key))
}
