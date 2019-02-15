package kvbackend

import (
	"context"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/func/func/storage"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

// Bolt stores key-value pairs in bolt db.
type Bolt struct {
	db *bolt.DB
}

// NewBolt creates a new BoltDB instance with the default location
// (~/.func/state.db). The directory is created if it does not exist.
func NewBolt() (*Bolt, error) {
	u, err := user.Current()
	if err != nil {
		return nil, errors.Wrap(err, "get user")
	}
	file := filepath.Join(u.HomeDir, ".func", "state.db")
	return NewBoltWithFile(file)
}

// NewBoltWithFile creates and opens a database at the given path. If the file
// or directory do not exist, they are created.
func NewBoltWithFile(file string) (*Bolt, error) {
	if err := os.MkdirAll(filepath.Dir(file), 0750); err != nil {
		return nil, errors.Wrapf(err, "ensure dir exists: %s", filepath.Dir(file))
	}
	db, err := bolt.Open(file, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return nil, errors.Wrap(err, "open bolt db")
	}
	return &Bolt{db: db}, nil
}

// Close closes the Bolt DB store and releases all resources.
func (b *Bolt) Close() error {
	return b.db.Close()
}

// Put creates or updates a value.
func (b *Bolt) Put(ctx context.Context, key string, value []byte) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		buc, k, err := boltBucketKey(key)
		if err != nil {
			return errors.Wrap(err, "get bucket name")
		}
		b, err := tx.CreateBucketIfNotExists(buc)
		if err != nil {
			return errors.Wrap(err, "ensure bucket exists")
		}
		return b.Put(k, value)
	})
}

// Get returns a single value.
func (b *Bolt) Get(ctx context.Context, key string) ([]byte, error) {
	var ret []byte
	if err := b.db.View(func(tx *bolt.Tx) error {
		buc, k, err := boltBucketKey(key)
		if err != nil {
			return errors.Wrap(err, "get bucket name")
		}
		b := tx.Bucket(buc)
		if b == nil {
			return storage.ErrNotFound
		}
		data := b.Get(k)
		if len(data) == 0 {
			return storage.ErrNotFound
		}
		ret = make([]byte, len(data))
		copy(ret, data)
		return nil
	}); err != nil {
		return nil, err
	}
	return ret, nil
}

// Delete deletes a key.
func (b *Bolt) Delete(ctx context.Context, key string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		buc, k, err := boltBucketKey(key)
		if err != nil {
			return errors.Wrap(err, "get bucket name")
		}
		b := tx.Bucket(buc)
		if b == nil {
			return storage.ErrNotFound
		}
		data := b.Get(k)
		if len(data) == 0 {
			return storage.ErrNotFound
		}
		if err = b.Delete(k); err != nil {
			return errors.Wrap(err, "delete key")
		}
		return nil
	})
}

// Scan performs a prefix scan and populates the returned map with any values
// matching the prefix.
//
// Note: the prefix must match a bucket. The bucket used is the key up to the
// last / character.
func (b *Bolt) Scan(ctx context.Context, prefix string) (map[string][]byte, error) {
	if strings.HasSuffix(prefix, "/") {
		return nil, errors.New("prefix should not contain trailing /")
	}
	ret := make(map[string][]byte)
	err := b.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(prefix))
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error { // nolint: unparam
			val := make([]byte, len(v))
			copy(val, v)
			key := prefix + "/" + string(k)
			ret[key] = val
			return nil
		})
	})
	return ret, err
}

// boltBucket returns the bucket and key to use for a storing a user specified
// key.
//
// The bucket is determined by looking for the last /. Anything before it is
// used as the bucket and anything after it as the key.
//
//   foo/bar/baz
//   ->
//   bucket: foo/bar
//   key:    baz
//
// Returns an error if the input does not contain a slash.
func boltBucketKey(input string) (bucket, key []byte, err error) {
	if strings.HasPrefix(input, "/") {
		return nil, nil, errors.New("input cannot start with a slash")
	}
	if strings.HasSuffix(input, "/") {
		return nil, nil, errors.New("input cannot end with a slash")
	}
	slash := strings.LastIndex(input, "/")
	if slash == -1 {
		return nil, nil, errors.New("input does not contain a slash")
	}
	return []byte(input[:slash]), []byte(input[slash+1:]), nil
}
