package taskpoet

import (
	"path"

	"github.com/jameycribbs/hare"
	"github.com/jameycribbs/hare/datastores/disk"
	"github.com/mitchellh/go-homedir"
)

type DBConfig struct {
	Path string
}

func GetHareDB(c *DBConfig) (*hare.Database, error) {
	var dbPath string
	if c.Path == "" {
		h, _ := homedir.Dir()
		dbPath = path.Join(h, ".taskpoet.db")

	} else {
		dbPath = c.Path
	}

	// Make a full path
	dbDataPath := path.Join(dbPath, "data")
	MakeDirectory(dbDataPath)

	// Disk engine
	ds, err := disk.New(dbDataPath, ".json")
	if err != nil {
		return nil, err
	}

	// New database is here
	db, err := hare.New(ds)
	if err != nil {
		return nil, err
	}
	return db, nil

}

/*func InitHareDB(c *DBConfig) error {
	var dbPath string
	if c.Path == "" {
		h, _ := homedir.Dir()
		dbPath = path.Join(h, ".taskpoet.db")

	} else {
		dbPath = c.Path
	}

	// Make a full path
	dbDataPath := path.Join(dbPath, "data")
	MakeDirectory(dbDataPath)

	// Disk engine
	ds, err := disk.New(dbDataPath, ".json")
	if err != nil {
		return err
	}

	// New database is here
	db, err := hare.New(ds)
	if err != nil {
		return err
	}
	defer db.Close()

	if !db.TableExists("tasks") {
		err = db.CreateTable("tasks")
		if err != nil {
			return err
		}

		t := &Task{
			Description: "Explore around in TaskPoet!",
		}
		now := time.Now()
		dueDuration, _ := ParseDuration("1w")
		due := now.Add(dueDuration)

		_, err := NewTask(*db, t, &Task{Due: due})
		if err != nil {
			return err
		}
	}

	return nil
}
*/
