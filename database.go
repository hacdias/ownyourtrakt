package main

import (
	"encoding/json"

	bolt "go.etcd.io/bbolt"
)

type database struct {
	db *bolt.DB
}

func newDatabase(path string) (*database, error) {
	db, err := bolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}

	return &database{
		db: db,
	}, nil
}

func (d *database) get(profileURL string) (*user, error) {
	u := &user{}

	err := d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}

		v := b.Get([]byte(profileURL))

		return json.Unmarshal(v, u)
	})

	return u, err
}

func (d *database) getAll() ([]*user, error) {
	users := []*user{}

	err := d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}

		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			u := &user{}
			err = json.Unmarshal(v, u)
			if err != nil {
				return err
			}
			users = append(users, u)
		}

		return nil
	})

	return users, err
}

func (d *database) save(u *user) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}

		encoded, err := json.Marshal(u)
		if err != nil {
			return err
		}

		return b.Put([]byte(u.ProfileURL), encoded)
	})
}

func (d *database) close() error {
	return d.db.Close()
}
