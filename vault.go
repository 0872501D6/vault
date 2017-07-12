package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// KeyIdProvider provides the openpgp key id
// The key id can be 64 bit long in hex or last 32 long bit in hex
type KeyIdProvider interface {
	key() string
}

// PassphraseProvider provides the pass phrase in byte slice
type PassphraseProvider interface {
	pass() []byte
}

// PgpProvider provides OpenPGP Key id, and if it's private, the passphrase
type PgpProvider interface {
	KeyIdProvider
	PassphraseProvider
}

// PublicPgpInfo contains OpenPGP Key id only
type PublicPgpInfo struct {
	keyId string
}

func (info PublicPgpInfo) key() string {
	return info.keyId
}

func (info PublicPgpInfo) pass() []byte {
	return []byte{}
}

func NewPublicPgpInfo(keyId string) PublicPgpInfo {
	return PublicPgpInfo{keyId: keyId}
}

// PrivatePgpInfo contains both Key id and passphrase
type PrivatePgpInfo struct {
	keyId      string
	passphrase []byte
}

func (info PrivatePgpInfo) key() string {
	return info.keyId
}

func (info PrivatePgpInfo) pass() []byte {
	return info.passphrase
}

func NewPrivatePgpInfo(keyId string, passphrase []byte) PrivatePgpInfo {
	return PrivatePgpInfo{keyId: keyId, passphrase: passphrase}
}

type Vault struct {
	directory string
}

func (v *Vault) baseDirectory() string {
	return v.directory
}

type NoVaultFoundError struct {
	dir string
}

func (e *NoVaultFoundError) Error() string {
	return fmt.Sprintf("No vault found for directory: %s", e.dir)
}

// If the path does not exists, we look for it one level up
func recursiveDirExists(pathArray []string) (string, bool) {
	if len(pathArray) == 0 {
		return "", false
	}
	dirPath := strings.Join(pathArray, "/")
	filePath := dirPath + "/" + CONF_DIR
	if dirExists(filePath) {
		return dirPath, true
	}
	return recursiveDirExists(pathArray[:len(pathArray)-1])
}

// setBaseDirectory sets the base vault directory for the vault object
// If no vault is found, return no vault found error
//func (v *Vault) setBaseDirectory() error {
func NewVault() (Vault, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Vault{}, err
	}
	pathTokens := strings.Split(cwd, "/")
	dir, exists := recursiveDirExists(pathTokens)
	if !exists {
		return Vault{}, &NoVaultFoundError{dir: cwd}
	}
	v := Vault{directory: dir}

	return v, nil
}

// LocalContext must be provided when files are added for encryption and saved
// as cached files
type LocalContext struct {
	vault *Vault
	pgp   PgpProvider
}

func (ctx *LocalContext) baseDirectory() string {
	return ctx.vault.baseDirectory()
}

func (ctx LocalContext) key() string {
	return ctx.pgp.key()
}

func (ctx LocalContext) pass() []byte {
	return ctx.pgp.pass()
}

type getPassphrase func() []byte

// NewLocalContext creates a new add context object
// If the add operation requires private key, then mark private as true
// f is the function to get passphrase
func NewLocalContext(private bool, f getPassphrase) LocalContext {
	v, err := NewVault()
	if err != nil {
		log.Fatal(err.Error())
	}
	configPath := makePath(v.baseDirectory(), CONF_DIR, CONFIG)
	confMap := ReadConfig(configPath)
	var pgpProvider PgpProvider
	keyId := confMap["signingkey"]
	if private {
		passphrase := f()
		pgpProvider = NewPrivatePgpInfo(keyId, passphrase)
	} else {
		pgpProvider = NewPublicPgpInfo(keyId)
	}
	addContext := LocalContext{
		vault: &v,
		pgp:   pgpProvider,
	}
	return addContext
}

// AWSContext must be provided for operation Push and Fetch
// It provides vault directory information and aws credentials
type AWSContext struct {
	dir       string // the local dir of vault config
	region    string // region code http://docs.aws.amazon.com/general/latest/gr/rande.html  example: us-east-1
	key       string // aws access key id
	sec       string // aws secret access key
	remoteDir string // namely the bucket for s3, and vault for glacier
}

func (aws AWSContext) baseDirectory() string {
	return aws.dir
}

func (aws AWSContext) awsRegion() string {
	return aws.region
}

func (aws AWSContext) accessKey() string {
	return aws.key
}

func (aws AWSContext) secret() string {
	return aws.sec
}

func (aws AWSContext) remote() string {
	return aws.remoteDir
}

// NewAWSContext creates a new context for AWS operations
func NewAWSContext() AWSContext {
	v, err := NewVault()
	if err != nil {
		log.Fatal(err.Error())
	}

	credPath := makePath(v.baseDirectory(), CONF_DIR, CRED)
	credMap := ReadConfig(credPath)
	key := credMap["aws_access_key_id"]
	sec := credMap["aws_secret_access_key"]
	confPath := makePath(v.baseDirectory(), CONF_DIR, CONFIG)
	configMap := ReadConfig(confPath)
	region := configMap["region"]
	remote := configMap["remote"]
	return AWSContext{
		dir:       v.baseDirectory(),
		region:    region,
		key:       key,
		sec:       sec,
		remoteDir: remote,
	}
}
