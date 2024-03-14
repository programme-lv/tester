package storage

import (
	"sync"
)

type Storage struct {
	mu  sync.Mutex
	dir string
}

var instance *Storage

var once sync.Once

func GetInstance() (*Storage, error) {
	if instance == nil {
		var fErr error
		once.Do(func() {
			instance = &Storage{}

			dir, err := getUserCacheDir()
			if err != nil {
				fErr = err
				return
			}
			instance.dir = dir
		})
		if fErr != nil {
			return nil, fErr
		}
	}
	return instance, nil
}
