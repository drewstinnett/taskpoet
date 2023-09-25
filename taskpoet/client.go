/*
Package taskpoet is the main worker library
*/
package taskpoet

import (
	"errors"
	"io"
	"path"

	"github.com/mitchellh/go-homedir"
	bolt "go.etcd.io/bbolt"
)

// Option helper for functional options with error reporting
type Option func() (func(*Poet), error)

func success(opt func(*Poet)) Option {
	return func() (func(*Poet), error) {
		return opt, nil
	}
}

func failure(err error) Option {
	return func() (func(*Poet), error) {
		return nil, err
	}
}

// New returns a new poet object and optional error
func New(options ...Option) (*Poet, error) {
	p := &Poet{}
	for _, option := range options {
		opt, err := option()
		if err != nil {
			return nil, err
		}
		opt(p)
	}
	return p, nil
}

// DBConfig configures the database
type DBConfig struct {
	Path      string
	Namespace string
}

// WithNamespace passes a namespace in to the new Poet object
func WithNamespace(n string) Option {
	if n == "" {
		return failure(errors.New("namespace cannot be empty"))
	}
	return success(func(p *Poet) {
		p.Namespace = n
	})
}

// Poet isi the main operator for this whole thing
type Poet struct {
	DB        *bolt.DB
	Namespace string
	Task      TaskService
}

// NewLocalClient returns a new poet the old way
// Deprecated: Use New with functional options instead
func NewLocalClient(c *DBConfig) (*Poet, error) {
	var dbPath string
	if c.Path == "" {
		h, _ := homedir.Dir()
		dbPath = path.Join(h, ".taskpoet.db")
	} else {
		dbPath = c.Path
	}

	var namespace string
	if c.Namespace == "" {
		namespace = "default"
	} else {
		namespace = c.Namespace
	}

	// Make a full path
	db, err := bolt.Open(dbPath, 0o600, nil)
	// defer db.Close()
	if err != nil {
		return nil, err
	}
	lc := Poet{
		DB:        db,
		Namespace: namespace,
	}
	lc.Task = &TaskServiceOp{
		localClient: &lc,
	}
	// New database is here
	return &lc, nil
}

// InitDB initializes the database
func InitDB(c *DBConfig) error {
	localClient, err := NewLocalClient(c)
	if err != nil {
		return err
	}

	// store some data
	err = localClient.DB.Update(func(tx *bolt.Tx) error {
		// localClient.
		bucket := tx.Bucket([]byte(localClient.Task.BucketName()))
		if bucket == nil {
			_, berr := tx.CreateBucket([]byte(localClient.Task.BucketName()))
			if berr != nil {
				return berr
			}
		}
		return nil
	})
	defer dclose(localClient.DB)
	if err != nil {
		return err
	}

	return nil
}

func dclose(c io.Closer) {
	err := c.Close()
	if err != nil {
		panic(err)
	}
}
