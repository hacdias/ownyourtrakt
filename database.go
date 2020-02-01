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

func (d *database) getUser(domain string) (*user, error) {
	u := &user{}

	err := d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}

		v := b.Get([]byte(domain))

		return json.Unmarshal(v, u)
	})

	return u, err
}

func (d *database) saveUser(u *user) error {
	return d.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}

		encoded, err := json.Marshal(u)
		if err != nil {
			return err
		}

		return b.Put([]byte(u.Domain), encoded)
	})
}

func (d *database) close() error {
	return d.db.Close()
}
