package main

import (
	"golang.org/x/crypto/openpgp/packet"
)

func AddCache(ctx *LocalContext, fns []string) []string {
	pathList := []string{}
	if len(fns) == 0 {
		return pathList
	}
	defaultConfig := &packet.Config{
		DefaultCompressionAlgo: 1,
		CompressionConfig:      &packet.CompressionConfig{Level: 5},
	}
	baseDir := ctx.baseDirectory()
	cacheDir := makePath(baseDir, CONF_DIR, CACHE)
	dbDir := makePath(baseDir, CONF_DIR, DB)

	for _, fn := range fns {
		fullPath := makePath(baseDir, fn)
		digest, path := EncryptFile(ctx, fn, cacheDir, defaultConfig)
		pathList = append(pathList, path)
		kv := LoadBadger(dbDir)
		vf, err := getVaultFile(kv, digest)
		if err != nil {
			nvf := VaultFile{
				Hash:    digest,
				Aliases: []string{fullPath},
				Glacier: "",
				KeyId:   ctx.key(),
			}
			insertVaultFile(kv, digest, nvf)
		} else {
			for _, p := range vf.Aliases {
				if p == fullPath {
					continue
				}
			}
			vf.Aliases = append(vf.Aliases, fullPath)
			insertVaultFile(kv, digest, vf)
		}
	}

	return pathList
}
