package taskpoet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	bolt "go.etcd.io/bbolt"
)

/*
Storage layout. Everything for a namespace lives under one top level bucket
"ns/<namespace>", which holds these nested buckets:

	meta        schema_version
	tasks       <uuid>                    -> JSON Task
	idx_status  <status>/<uuid>           -> 1
	idx_parent  <parent uuid>/<uuid>      -> 1   (recurrence instances)
	idx_blocks  <depends on uuid>/<uuid>  -> 1   (reverse dependencies)

Tasks are keyed by UUID alone, and keys are sorted, so a partial ID lookup is
a cursor seek instead of a scan. The indexes are only ever written by putTask,
inside the same transaction as the task itself, so they cannot drift.
*/

const schemaVersion = 2

var (
	bucketMeta      = []byte("meta")
	bucketTasks     = []byte("tasks")
	bucketIdxStatus = []byte("idx_status")
	bucketIdxParent = []byte("idx_parent")
	bucketIdxBlocks = []byte("idx_blocks")
	keySchema       = []byte("schema_version")
	indexValue      = []byte{1}
)

var (
	// ErrNotFound is returned when no task matches
	ErrNotFound = errors.New("task not found")
	// ErrExists is returned when adding a task that is already there
	ErrExists = errors.New("task already exists")
	// ErrAmbiguous is returned when a partial ID matches more than one task
	ErrAmbiguous = errors.New("ambiguous task id")
	// ErrLegacyDatabase is returned when the file was written by taskpoet v0.x
	ErrLegacyDatabase = errors.New("this is a taskpoet v0.x database, which v2 cannot read: move it aside and re-import from Taskwarrior with 'taskpoet import taskwarrior'")
	// ErrNewerSchema is returned when the file was written by a newer taskpoet
	ErrNewerSchema = errors.New("database was written by a newer version of taskpoet")
	// ErrLocked is returned when another process has the database open
	ErrLocked = errors.New("database is locked by another taskpoet process")
)

// Store is the bbolt backed storage for tasks
type Store struct {
	db   *bolt.DB
	root []byte
	path string
	now  func() time.Time
}

// StoreOption is a functional option for OpenStore
type StoreOption func(*Store)

// WithClock overrides the clock, mostly useful for tests
func WithClock(now func() time.Time) StoreOption {
	return func(s *Store) {
		s.now = now
	}
}

// OpenStore opens (and initializes if needed) the database at path
func OpenStore(path, namespace string, options ...StoreOption) (*Store, error) {
	if namespace == "" {
		return nil, errors.New("namespace cannot be empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		if errors.Is(err, bolt.ErrTimeout) {
			return nil, fmt.Errorf("%w: %v", ErrLocked, path)
		}
		return nil, err
	}
	s := &Store{
		db:   db,
		root: []byte("ns/" + namespace),
		path: path,
		now:  defaultNow,
	}
	for _, opt := range options {
		opt(s)
	}
	if err := s.init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func defaultNow() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}

// Close closes the database
func (s *Store) Close() error {
	return s.db.Close()
}

// Path returns the path of the database file
func (s *Store) Path() string {
	return s.path
}

func (s *Store) init() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		// v0.x kept one top level bucket per namespace, named /<ns>/tasks
		var legacy bool
		if err := tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			if bytes.HasPrefix(name, []byte("/")) {
				legacy = true
			}
			return nil
		}); err != nil {
			return err
		}
		if legacy {
			return ErrLegacyDatabase
		}

		root := tx.Bucket(s.root)
		if root == nil {
			var err error
			if root, err = tx.CreateBucket(s.root); err != nil {
				return err
			}
			meta, err := root.CreateBucket(bucketMeta)
			if err != nil {
				return err
			}
			if err := meta.Put(keySchema, []byte(strconv.Itoa(schemaVersion))); err != nil {
				return err
			}
		}

		meta := root.Bucket(bucketMeta)
		if meta == nil {
			return fmt.Errorf("namespace %q has no metadata, the database looks corrupt", s.root)
		}
		v, err := strconv.Atoi(string(meta.Get(keySchema)))
		if err != nil {
			return fmt.Errorf("unreadable schema version: %w", err)
		}
		if v > schemaVersion {
			return fmt.Errorf("%w (schema %d, this build understands %d)", ErrNewerSchema, v, schemaVersion)
		}

		for _, name := range [][]byte{bucketTasks, bucketIdxStatus, bucketIdxParent, bucketIdxBlocks} {
			if _, err := root.CreateBucketIfNotExists(name); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *Store) bucket(tx *bolt.Tx, name []byte) *bolt.Bucket {
	return tx.Bucket(s.root).Bucket(name)
}

func idxKey(a, b string) []byte {
	return []byte(a + "/" + b)
}

func decodeTask(v []byte) (*Task, error) {
	var t Task
	if err := json.Unmarshal(v, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Store) getTask(tx *bolt.Tx, id string) (*Task, error) {
	v := s.bucket(tx, bucketTasks).Get([]byte(id))
	if v == nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, id)
	}
	return decodeTask(v)
}

// putTask writes the task and brings every index in line with it. All writes
// go through here.
func (s *Store) putTask(tx *bolt.Tx, t *Task) error {
	if err := t.Validate(); err != nil {
		return err
	}
	old, err := s.getTask(tx, t.UUID)
	switch {
	case err == nil:
		if err := s.unindex(tx, old); err != nil {
			return err
		}
	case !errors.Is(err, ErrNotFound):
		return err
	}

	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	if err := s.bucket(tx, bucketTasks).Put([]byte(t.UUID), b); err != nil {
		return err
	}
	return s.index(tx, t)
}

func (s *Store) unindex(tx *bolt.Tx, t *Task) error {
	if err := s.bucket(tx, bucketIdxStatus).Delete(idxKey(string(t.Status), t.UUID)); err != nil {
		return err
	}
	if t.Parent != "" {
		if err := s.bucket(tx, bucketIdxParent).Delete(idxKey(t.Parent, t.UUID)); err != nil {
			return err
		}
	}
	for _, d := range t.Depends {
		if err := s.bucket(tx, bucketIdxBlocks).Delete(idxKey(d, t.UUID)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) index(tx *bolt.Tx, t *Task) error {
	if err := s.bucket(tx, bucketIdxStatus).Put(idxKey(string(t.Status), t.UUID), indexValue); err != nil {
		return err
	}
	if t.Parent != "" {
		if err := s.bucket(tx, bucketIdxParent).Put(idxKey(t.Parent, t.UUID), indexValue); err != nil {
			return err
		}
	}
	for _, d := range t.Depends {
		if err := s.bucket(tx, bucketIdxBlocks).Put(idxKey(d, t.UUID), indexValue); err != nil {
			return err
		}
	}
	return nil
}

// scanPrefix calls fn for every key in b that starts with prefix
func scanPrefix(b *bolt.Bucket, prefix string, fn func(k, v []byte) error) error {
	p := []byte(prefix)
	c := b.Cursor()
	for k, v := c.Seek(p); k != nil && bytes.HasPrefix(k, p); k, v = c.Next() {
		if err := fn(k, v); err != nil {
			return err
		}
	}
	return nil
}

// idsFromIndex returns the uuid half of every '<prefix>/<uuid>' key in an index
func idsFromIndex(b *bolt.Bucket, prefix string) ([]string, error) {
	var ids []string
	err := scanPrefix(b, prefix+"/", func(k, _ []byte) error {
		ids = append(ids, string(k[len(prefix)+1:]))
		return nil
	})
	return ids, err
}

// Get returns a task from its exact UUID
func (s *Store) Get(id string) (*Task, error) {
	var t *Task
	err := s.db.View(func(tx *bolt.Tx) error {
		var err error
		t, err = s.getTask(tx, id)
		return err
	})
	return t, err
}

// GetByPrefix returns the one task whose UUID starts with prefix. If statuses
// are given, only tasks in one of those statuses are considered.
func (s *Store) GetByPrefix(prefix string, statuses ...Status) (*Task, error) {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		return nil, errors.New("task id cannot be empty")
	}
	var matches Tasks
	if err := s.db.View(func(tx *bolt.Tx) error {
		return scanPrefix(s.bucket(tx, bucketTasks), prefix, func(_, v []byte) error {
			t, err := decodeTask(v)
			if err != nil {
				return err
			}
			if len(statuses) == 0 || hasStatus(statuses, t.Status) {
				matches = append(matches, t)
			}
			return nil
		})
	}); err != nil {
		return nil, err
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: nothing matches %q", ErrNotFound, prefix)
	case 1:
		return matches[0], nil
	}
	ids := make([]string, len(matches))
	for i, m := range matches {
		ids[i] = m.UUID
	}
	return nil, fmt.Errorf("%w: %q matches %d tasks (%v), use more of the id", ErrAmbiguous, prefix, len(matches), strings.Join(ids, ", "))
}

func hasStatus(l []Status, s Status) bool {
	for _, c := range l {
		if c == s {
			return true
		}
	}
	return false
}

// List returns every task in one of the given statuses, or all tasks when no
// statuses are given. The order is not defined, sort the result.
func (s *Store) List(statuses ...Status) (Tasks, error) {
	var tasks Tasks
	err := s.db.View(func(tx *bolt.Tx) error {
		tb := s.bucket(tx, bucketTasks)
		if len(statuses) == 0 {
			return tb.ForEach(func(_, v []byte) error {
				t, err := decodeTask(v)
				if err != nil {
					return err
				}
				tasks = append(tasks, t)
				return nil
			})
		}
		for _, st := range statuses {
			ids, err := idsFromIndex(s.bucket(tx, bucketIdxStatus), string(st))
			if err != nil {
				return err
			}
			for _, id := range ids {
				t, err := s.getTask(tx, id)
				if err != nil {
					return err
				}
				tasks = append(tasks, t)
			}
		}
		return nil
	})
	return tasks, err
}

// Instances returns the tasks spawned from a recurrence template
func (s *Store) Instances(parent string) (Tasks, error) {
	return s.tasksFromIndex(bucketIdxParent, parent)
}

// Blocking returns the tasks that list id in their Depends, whatever their status
func (s *Store) Blocking(id string) (Tasks, error) {
	return s.tasksFromIndex(bucketIdxBlocks, id)
}

func (s *Store) tasksFromIndex(index []byte, key string) (Tasks, error) {
	var tasks Tasks
	err := s.db.View(func(tx *bolt.Tx) error {
		var err error
		tasks, err = s.indexTasks(tx, index, key)
		return err
	})
	return tasks, err
}

// indexTasks returns the tasks listed under key in an index, inside a transaction
func (s *Store) indexTasks(tx *bolt.Tx, index []byte, key string) (Tasks, error) {
	ids, err := idsFromIndex(s.bucket(tx, index), key)
	if err != nil {
		return nil, err
	}
	tasks := make(Tasks, 0, len(ids))
	for _, id := range ids {
		t, err := s.getTask(tx, id)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

// CountByStatus returns how many tasks are in each status
func (s *Store) CountByStatus() (map[Status]int, error) {
	counts := map[Status]int{}
	err := s.db.View(func(tx *bolt.Tx) error {
		return s.bucket(tx, bucketIdxStatus).ForEach(func(k, _ []byte) error {
			status, _, _ := strings.Cut(string(k), "/")
			counts[Status(status)]++
			return nil
		})
	})
	return counts, err
}

// Put writes a task, replacing any existing task with the same UUID. The
// Modified time is left alone, so imports keep the original.
func (s *Store) Put(t *Task) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return s.putTask(tx, t)
	})
}

// Add adds a new task, filling in the UUID and timestamps if they are missing
func (s *Store) Add(t *Task) error {
	now := s.now()
	if t.UUID == "" {
		t.UUID = uuid.New().String()
	}
	if t.Status == "" {
		t.Status = StatusPending
	}
	if t.Entry.IsZero() {
		t.Entry = now
	}
	if t.Modified.IsZero() {
		t.Modified = now
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		if _, err := s.getTask(tx, t.UUID); err == nil {
			return fmt.Errorf("%w: %v", ErrExists, t.UUID)
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		return s.putTask(tx, t)
	})
}

// Update reads a task, lets fn change it, and writes it back, stamping Modified
func (s *Store) Update(id string, fn func(*Task) error) (*Task, error) {
	var out *Task
	err := s.db.Update(func(tx *bolt.Tx) error {
		t, err := s.getTask(tx, id)
		if err != nil {
			return err
		}
		if err := fn(t); err != nil {
			return err
		}
		t.Modified = s.now()
		out = t
		return s.putTask(tx, t)
	})
	return out, err
}

// Complete marks a task as completed
func (s *Store) Complete(id string) (*Task, error) {
	return s.transition(id, StatusCompleted)
}

// Delete marks a task as deleted. The record is kept, like Taskwarrior does.
func (s *Store) Delete(id string) (*Task, error) {
	return s.transition(id, StatusDeleted)
}

func (s *Store) transition(id string, to Status) (*Task, error) {
	var out *Task
	err := s.db.Update(func(tx *bolt.Tx) error {
		t, err := s.getTask(tx, id)
		if err != nil {
			return err
		}
		switch {
		case t.Status == to:
			return fmt.Errorf("task %v is already %v", t.ShortID(), to)
		case t.Status == StatusRecurring && to == StatusCompleted:
			return fmt.Errorf("task %v is a recurrence template, it can only be deleted", t.ShortID())
		}
		from, now := t.Status, s.now()
		t.Status = to
		t.End = &now
		t.Modified = now
		if err := s.putTask(tx, t); err != nil {
			return err
		}
		out = t
		return s.afterTransition(tx, t, from)
	})
	return out, err
}

// afterTransition is where a status change ripples out to other tasks
func (s *Store) afterTransition(tx *bolt.Tx, t *Task, from Status) error {
	// A template that is deleted takes its pending instances with it
	if from == StatusRecurring && t.Status == StatusDeleted {
		return s.deleteInstances(tx, t)
	}
	if t.Parent == "" {
		return nil
	}

	// Keep the template's mask in step with its instances
	parent, err := s.getTask(tx, t.Parent)
	if errors.Is(err, ErrNotFound) {
		return nil
	} else if err != nil {
		return err
	}
	if t.IMask != nil {
		char := byte('-')
		switch t.Status {
		case StatusCompleted:
			char = '+'
		case StatusDeleted:
			char = 'X'
		}
		parent.Mask = setMaskChar(parent.Mask, *t.IMask, char)
		if err := s.putTask(tx, parent); err != nil {
			return err
		}
	}

	// Finishing a chained instance starts the next one
	if t.Status == StatusCompleted {
		return s.spawnChainedAfter(tx, parent.UUID)
	}
	return nil
}

// setMaskChar sets position i of a recurrence mask, padding with '-' as needed
func setMaskChar(mask string, i int, c byte) string {
	b := []byte(mask)
	for len(b) <= i {
		b = append(b, '-')
	}
	b[i] = c
	return string(b)
}
