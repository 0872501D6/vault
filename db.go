package main

import (
	"encoding/json"
	"log"
)

import (
	"github.com/dgraph-io/badger"
)

type VaultFile struct {
	Hash    string   `json:"hash"`    // raw hash value computed by glacier SHA256 tree hasher
	Aliases []string `json:"aliases"` // all file path relevant to the vault config path
	Glacier string   `json:"glacier"` // glacier id
	KeyId   string   `json:"keyid"`   // openpgp key id, last 32 bit in hex
}

// Create or get the badger KV object
func LoadBadger(path string) *badger.KV {
	opt := badger.DefaultOptions
	opt.Dir = path
	opt.ValueDir = path
	kv, err := badger.NewKV(&opt)
	if err != nil {
		log.Fatal("Fail to create a new badger KV")
	}
	return kv
}

// Insert new record to kv, override the previous one if exists
func insertVaultFile(kv *badger.KV, key string, v VaultFile) {
	kb := []byte(key)
	valueJson, err := json.Marshal(&v)
	if err != nil {
		log.Fatal("Error encoding struct as json")
	}
	vb := []byte(string(valueJson))
	kv.Set(kb, vb)
}

// Get VaultFile object by key
func getVaultFile(kv *badger.KV, key string) (VaultFile, error) {
	var item badger.KVItem

	err := kv.Get([]byte(key), &item)
	if err != nil {
		return VaultFile{}, err
	}
	vf := VaultFile{}
	err = json.Unmarshal(item.Value(), &vf)
	if err != nil {
		return VaultFile{}, err
	}
	return vf, nil
}
