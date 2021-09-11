package taskpoet

import (
	"path"

	"github.com/mitchellh/go-homedir"
	bolt "go.etcd.io/bbolt"
)

type DBConfig struct {
	Path      string
	Namespace string
}

type LocalClient struct {
	DB        *bolt.DB
	Namespace string
	Task      TaskService
}

func NewLocalClient(c *DBConfig) (*LocalClient, error) {
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
	db, err := bolt.Open(dbPath, 0600, nil)
	//defer db.Close()
	if err != nil {
		return nil, err
	}
	lc := LocalClient{
		DB:        db,
		Namespace: namespace,
	}
	lc.Task = &TaskServiceOp{
		localClient: &lc,
	}
	// New database is here
	return &lc, nil

}

func InitDB(c *DBConfig) error {
	localClient, err := NewLocalClient(c)
	if err != nil {
		return err
	}

	// store some data
	err = localClient.DB.Update(func(tx *bolt.Tx) error {
		//localClient.
		bucket := tx.Bucket([]byte(localClient.Task.BucketName()))
		if bucket == nil {
			_, err := tx.CreateBucket([]byte(localClient.Task.BucketName()))
			if err != nil {
				return err
			}
		}
		return nil

	})
	defer localClient.DB.Close()
	if err != nil {
		return err
	}

	return nil
}
