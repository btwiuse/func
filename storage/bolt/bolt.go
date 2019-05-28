package bolt

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/func/func/resource"
	"github.com/pkg/errors"
	bolt "go.etcd.io/bbolt"
)

// The ResourceCodec encodes resources for storage.
type ResourceCodec interface {
	MarshalResource(resource.Resource) ([]byte, error)
	UnmarshalResource(b []byte) (resource.Resource, error)
}

// Bolt stores key-value pairs in bolt db.
type Bolt struct {
	db    *bolt.DB
	codec ResourceCodec
}

// DefaultFile returns the default file to use for the file on disk.
func DefaultFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.Wrap(err, "get home dir")
	}
	return filepath.Join(home, ".func", "state.db"), nil
}

// New creates and opens a database at the given file.
// If the file or directory does not exist, it is created.
func New(file string, codec ResourceCodec) (*Bolt, error) {
	if err := os.MkdirAll(filepath.Dir(file), 0750); err != nil {
		return nil, errors.Wrapf(err, "ensure dir exists: %s", filepath.Dir(file))
	}
	db, err := bolt.Open(file, 0600, &bolt.Options{Timeout: 3 * time.Second})
	if err != nil {
		return nil, errors.Wrap(err, "open bolt db")
	}
	return &Bolt{db: db, codec: codec}, nil
}

// Close closes the Bolt DB store and releases all resources.
func (b *Bolt) Close() error {
	return b.db.Close()
}

// Put creates or updates a resource.
func (b *Bolt) Put(ctx context.Context, ns, project string, resource resource.Resource) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket, err := b.createBucketIfNotExists(tx, []string{ns, project, "resources"})
		if err != nil {
			return errors.Wrap(err, "ensure bucket")
		}
		k := []byte(resource.Name)
		data, err := b.codec.MarshalResource(resource)
		if err != nil {
			return errors.Wrap(err, "marshal resource")
		}
		return bucket.Put(k, data)
	})
}

// Delete deletes a resource. No-op if the resource does not exist.
func (b *Bolt) Delete(ctx context.Context, ns, project, name string) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		bucket := b.getBucket(tx, []string{ns, project, "resources"})
		if bucket == nil {
			return nil
		}
		return bucket.Delete([]byte(name))
	})
}

// List lists all resources in a project.
func (b *Bolt) List(ctx context.Context, ns, project string) (map[string]resource.Resource, error) {
	out := make(map[string]resource.Resource)
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket := b.getBucket(tx, []string{ns, project, "resources"})
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(k, v []byte) error {
			res, err := b.codec.UnmarshalResource(v)
			if err != nil {
				return err
			}
			name := string(k)
			out[name] = res
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// createBucketIfNotExists creates any buckets on the given path that do not
// exist, and returns the final bucket.
func (b *Bolt) createBucketIfNotExists(tx *bolt.Tx, path []string) (*bolt.Bucket, error) {
	if len(path) == 0 {
		panic("path is empty")
	}
	bucket, err := tx.CreateBucketIfNotExists([]byte(path[0]))
	if err != nil {
		return nil, errors.Wrap(err, "root bucket")
	}
	for _, p := range path[1:] {
		tmp, err := bucket.CreateBucketIfNotExists([]byte(p))
		if err != nil {
			return nil, errors.Wrapf(err, "part %s", p)
		}
		bucket = tmp
	}
	return bucket, nil
}

// getBucket gets a bucket at the given path. Returns nil if the bucket does not exist.
func (b *Bolt) getBucket(tx *bolt.Tx, path []string) *bolt.Bucket {
	if len(path) == 0 {
		panic("path is empty")
	}
	bucket := tx.Bucket([]byte(path[0]))
	for _, p := range path[1:] {
		if bucket == nil {
			break
		}
		bucket = bucket.Bucket([]byte(p))
	}
	return bucket
}
