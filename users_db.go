package main

import (
	"encoding/json"

	bolt "go.etcd.io/bbolt"
)

type usersDB struct {
	db *bolt.DB
}

func newUsersDB(path string) (*usersDB, error) {
	db, err := bolt.Open(path, 0666, nil)
	if err != nil {
		return nil, err
	}

	return &usersDB{
		db: db,
	}, nil
}

func (d *usersDB) get(domain string) (*user, error) {
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

func (d *usersDB) save(u *user) error {
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

func (d *usersDB) close() error {
	return d.db.Close()
}
