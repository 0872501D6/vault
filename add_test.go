package main

import (
	"os"
	"testing"
)

// newAddContextForTest creates a synthesised add context
func newPrivLocalContextForTest() LocalContext {
	v := Vault{"test_files"}
	keyId := "C21B7817"
	passphrase := []byte("b88d80170")

	pgp := PrivatePgpInfo{keyId: keyId, passphrase: passphrase}

	return LocalContext{vault: &v, pgp: pgp}
}

// newAddContextForTest creates a synthesised add context
func newPubLocalContextForTest() LocalContext {
	v := Vault{"test_files"}
	keyId := "C21B7817"

	pgp := PublicPgpInfo{keyId: keyId}

	return LocalContext{vault: &v, pgp: pgp}
}

func newPath() {
	cacheDir := "test_files/.vault/cache"
	_, err := os.Stat(cacheDir)
	if err != nil && os.IsNotExist(err) {
		os.MkdirAll(cacheDir, 0775)
	}
}

func newDb() {
	dbDir := "test_files/.vault/db"
	_, err := os.Stat(dbDir)
	if err != nil && os.IsNotExist(err) {
		os.MkdirAll(dbDir, 0775)
	}
	kv := LoadBadger(dbDir)
	defer kv.Close()
}

func TestAddCache(t *testing.T) {
	newPath()
	newDb()
	ctx := newPrivLocalContextForTest()
	pl := AddCache(&ctx, []string{"test_files/test_file"})
	if len(pl) != 1 || pl[0] != "test_files/.vault/cache/4cd23549dde14b6a1e1cd08501c599c9a86c098b6a96a15290fc78c237923f58" {
		t.Fatal("Add cache fails")
	}
	os.RemoveAll("test_files/.vault")
}
