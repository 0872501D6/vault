package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/packet"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

import (
	"github.com/aws/aws-sdk-go/aws"
)

func TestPrintEntity(t *testing.T) {
	keyId := "C21B7817"
	targetStr := "B2E225E7C21B7817\tb88d80170 test key 01 <b88d80170@gmail.com>"
	// public key
	if entityStr := printEntity(getEntityById(getPubKeyringDir(), keyId)); entityStr != targetStr {
		t.Fatal(entityStr)
	}
	// private key
	if entityStr := printEntity(getEntityById(getPrivKeyringDir(), keyId)); entityStr != targetStr {
		t.Fatal(entityStr)
	}
	// some random key, silently fail
	if entityStr := printEntity(getEntityById(getPrivKeyringDir(), "some random string")); entityStr != "" {
		t.Fatal(entityStr)
	}
}

func promptForKeyC21B7817() openpgp.PromptFunction {
	return func(keys []openpgp.Key, symm bool) ([]byte, error) {
		if symm {
			return nil, errors.ErrKeyIncorrect
		}
		if len(keys) == 0 {
			return nil, errors.ErrKeyIncorrect
		}
		entityList := getEntityList(getPrivKeyringDir())
		entity := filterEntityById("C21B7817", &entityList)
		passphrase := []byte("b88d80170")
		// key
		err := keys[0].PrivateKey.Decrypt(passphrase)
		if err != nil {
			return nil, errors.ErrKeyIncorrect
		}
		// subkeys
		for _, subkey := range entity.Subkeys {
			subkey.PrivateKey.Decrypt(passphrase)
		}
		return nil, nil
	}
}

func defaultConfig() *packet.Config {
	return &packet.Config{
		DefaultCompressionAlgo: 1,
		CompressionConfig:      &packet.CompressionConfig{Level: 5},
	}
}

func cleanFiles(encryptedFn string) {
	// clear files
	os.Remove(encryptedFn)
	os.Remove(encryptedFn + ".decrypt")
}

func encryptDecrypt(t *testing.T, ctx *LocalContext, fn string) {
	config := defaultConfig()
	ofp := "test_files"
	prompt := promptForKeyC21B7817()
	_, encryptedFn := EncryptFile(ctx, fn, ofp, config)
	decryptedFn := DecryptFile(encryptedFn, config, prompt)

	// compare
	fbOrig, _ := ioutil.ReadFile(fn)
	bodyOrig := io.ReadSeeker(bytes.NewReader(fbOrig))
	digestOrig := TreeHash(bodyOrig)

	fbDecrypt, _ := ioutil.ReadFile(decryptedFn)
	bodyDecrypt := aws.ReadSeekCloser(bytes.NewReader(fbDecrypt))
	digestDecrypt := TreeHash(bodyDecrypt)

	// clean dir
	cleanFiles(encryptedFn)

	if digestDecrypt != digestOrig {
		fatalMsg := fmt.Sprintf("Files don't match\tdecrypt:  %s,\toriginal: %s", digestDecrypt, digestOrig)
		t.Fatal(fatalMsg)
	}
}

func TestEncryptDecryptWithSign(t *testing.T) {
	ctx := newPrivLocalContextForTest()
	encryptDecrypt(t, &ctx, "test_files/test_file")
	encryptDecrypt(t, &ctx, "test_files/image.png")
}

func TestEncryptDecryptWithoutSign(t *testing.T) {
	ctx := newPubLocalContextForTest()
	encryptDecrypt(t, &ctx, "test_files/test_file")
	encryptDecrypt(t, &ctx, "test_files/image.png")
}

func TestVerifyWithSign(t *testing.T) {
	encryptVerify := func(t *testing.T, ctx *LocalContext, fn string) {
		config, ofp, prompt := defaultConfig(), "test_files", promptForKeyC21B7817()
		_, encryptedFn := EncryptFile(ctx, fn, ofp, config)
		sig, b := GetSignature(encryptedFn, config, prompt)
		if !b {
			t.Fatal("It should be signed")
		}
		sid := getShortKeyId(sig.SignedByKeyId)
		if !compareString(sid, "C21B7817") {
			t.Fatal("It should be signed by C21B7817, but by ", sid)
		}
		// clean dir
		cleanFiles(encryptedFn)
	}
	ctx := newPrivLocalContextForTest()
	encryptVerify(t, &ctx, "test_files/test_file")
	encryptVerify(t, &ctx, "test_files/image.png")
}

func TestVerifyWithoutSign(t *testing.T) {
	encryptVerify := func(t *testing.T, ctx *LocalContext, fn string) {
		config, ofp, prompt := defaultConfig(), "test_files", promptForKeyC21B7817()
		_, encryptedFn := EncryptFile(ctx, fn, ofp, config)
		_, b := GetSignature(encryptedFn, config, prompt)
		if b {
			t.Fatal("It should be not signed")
		}
		// clean dir
		cleanFiles(encryptedFn)
	}
	ctx := newPubLocalContextForTest()
	encryptVerify(t, &ctx, "test_files/test_file")
	encryptVerify(t, &ctx, "test_files/image.png")
}
