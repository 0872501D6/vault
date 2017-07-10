package main

import (
	"fmt"
	"log"
	"io/ioutil"
	"os"
)


func updateKV(vaultDir, key, glacierId string) error {
	dbPath := makePath(vaultDir, CONF_DIR, DB)
	kv := LoadBadger(dbPath)
	return updateVaultFileWithDigest(kv, key, glacierId)
}

func pushFiles(vaultDir string, ctx *AWSContext) {
	setAwsEnv(vaultDir)
	cacheFilePath := makePath(vaultDir, CONF_DIR, CACHE)
	svc := NewService(ctx.awsRegion())
	files, err := ioutil.ReadDir(cacheFilePath)
	if err != nil {
		log.Fatal("error access cache directory")
	}
	if len(files) == 0 {
		fmt.Println("Nothing to push")
	} else {
		fmt.Println("Start pushing")
	}
	for _, fi := range files {
		fn := makePath(cacheFilePath, fi.Name())
		output, err := UploadFile(fn, "TestVault", svc)
		if err != nil {
			continue // silently fail
		}
		glacierId := output.ArchiveId
		err = updateKV(vaultDir, fi.Name(), *glacierId)
		if err != nil {
			continue // silently fail
		}
		os.Remove(fn)
	}
}
