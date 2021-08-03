package taskpoet

import (
	"path"

	"github.com/mitchellh/go-homedir"
	bolt "go.etcd.io/bbolt"
)

type DBConfig struct {
	Path string
}

type LocalClient struct {
	DB   *bolt.DB
	Task TaskService
}

func NewLocalClient(c *DBConfig) (*LocalClient, error) {
	var dbPath string
	if c.Path == "" {
		h, _ := homedir.Dir()
		dbPath = path.Join(h, ".taskpoet.db")

	} else {
		dbPath = c.Path
	}

	// Make a full path
	db, err := bolt.Open(dbPath, 0600, nil)
	//defer db.Close()
	if err != nil {
		return nil, err
	}
	lc := LocalClient{
		DB: db,
	}
	lc.Task = &TaskServiceOp{localClient: &lc}
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
		bucket := tx.Bucket([]byte("tasks"))
		if bucket == nil {
			_, err := tx.CreateBucket([]byte("tasks"))
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
