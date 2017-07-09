package main

import (
	"os"
	"testing"
)

// Insert a new VaultFile and then get it
func TestInsert(t *testing.T) {
	hash := "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03"
	// use fake data
	vf := VaultFile{
		Hash:    hash,
		Aliases: []string{"foo/bar", "foo"},
		Glacier: "",
		KeyId:   "C21B7817",
	}

	kv := LoadBadger(os.TempDir())
	insertVaultFile(kv, hash, vf)
	vf, err := getVaultFile(kv, hash)

	if err != nil {
		t.Fatal("error insert or get: ", err.Error())
	}

	if vf.Hash != hash {
		t.Fatal("wrong hash value: ", vf.Hash)
	}
	if len(vf.Aliases) != 2 || vf.Aliases[0] != "foo/bar" || vf.Aliases[1] != "foo" {
		t.Fatal("wrong aliases")
	}
	if vf.Glacier != "" {
		t.Fatal("wrong glaicer id")
	}
	if vf.KeyId != "C21B7817" {
		t.Fatal("wrong keyid")
	}
}

// Insert a new VaultFile and then get a non-existing key
func TestInsert2(t *testing.T) {
	kv := LoadBadger(os.TempDir())
	vf := VaultFile{}
	insertVaultFile(kv, "1", vf)
	_, err := getVaultFile(kv, "2")
	if err == nil {
		t.Fatal("expect error here")
	}
}
