package main

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"strconv"
	"strings"
	"syscall"
)

type getDirFn func() string

func getHomeDir() string {
	currentUser, err := user.Current()
	if err != nil {
		log.Fatal("error getting current user")
	}
	return currentUser.HomeDir
}

func getPubKeyringDir() string {
	return fmt.Sprintf("%s/.gnupg/pubring.gpg", getHomeDir())
}

func getPrivKeyringDir() string {
	return fmt.Sprintf("%s/.gnupg/secring.gpg", getHomeDir())
}

func getPassphraseFromStdin() []byte {
	// enter passphrase
	fmt.Print("Please enter passphrase: ")
	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		log.Fatal("error getting password")
	}
	return bytePassword
}

func getEntityList(fn string) openpgp.EntityList {
	f, err := os.Open(fn)
	defer f.Close()
	if err != nil {
		log.Fatal("error opening key ring file")
	}
	entityList, err := openpgp.ReadKeyRing(f)
	if err != nil {
		log.Fatal("error reading key ring")
	}
	return entityList
}

// convert hex string to uint64
func string2uint64(s string) uint64 {
	num, err := strconv.ParseUint(s, 16, 64)
	if err != nil {
		log.Fatal("error converting uint64")
	}

	return num
}

func getKeysById(s string) []openpgp.Key {
	id := string2uint64(s)
	fn := getPubKeyringDir()
	el := getEntityList(fn)
	return el.KeysById(id)
}

// get the second half of the key id string
func getShortKeyId(id uint64) string {
	return getKeyId(id)[8:]
}

func getKeyId(id uint64) string {
	return strconv.FormatUint(id, 16)
}

func compareString(a, b string) bool {
	return strings.ToUpper(a) == strings.ToUpper(b)
}

func getEntityById(fn, keyId string) *openpgp.Entity {
	entityList := getEntityList(fn)
	for _, entity := range entityList {
		identities := entity.Identities
		for _, identity := range identities {
			id := getKeyId(*identity.SelfSignature.IssuerKeyId)
			if compareString(id, keyId) || compareString(id[8:], keyId) {
				return entity
			}
		}
	}
	return nil
}

func filterEntityById(keyId string, entityList *openpgp.EntityList) *openpgp.Entity {
	for _, entity := range *entityList {
		identities := entity.Identities
		for _, identity := range identities {
			id := getKeyId(*identity.SelfSignature.IssuerKeyId)
			if compareString(id, keyId) || compareString(id[8:], keyId) {
				return entity
			}
		}
	}
	return nil
}

func printEntity(entity *openpgp.Entity) string {
	if entity == nil {
		return ""
	}
	identities := entity.Identities
	for _, identity := range identities {
		keyId := strings.ToUpper(getKeyId(*identity.SelfSignature.IssuerKeyId))
		return fmt.Sprintf("%s\t%s", keyId, identity.Name)
	}
	return ""
}

// Encrypts the file and returns its sha256 hash value of the original file
// fn: file name to encrypt
// ofp: output file path
// entity: openpgp entity
// signed: if true, then it also signs the encryption with the same key
// config: encryption config
func encryptFileHelper(fn, ofp string, entity *openpgp.Entity, signed bool, config *packet.Config) (string, string) {
	entityList := []*openpgp.Entity{entity}

	br, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Fatal("error reading input file during encryption")
	}
	// obtain the hash value as file name
	body := io.ReadSeeker(bytes.NewReader(br))
	digest := TreeHash(body)
	digestPrefix := digest[:2]
	digestBody := digest[2:]
	// create folder if it does not exists
	_, err = os.Stat(ofp + "/" + digestPrefix)
	objectPath := fmt.Sprintf("%s/%s", digestPrefix, digestBody)
	if err != nil && os.IsNotExist(err) {
		os.Mkdir(ofp+"/"+digestPrefix, 0775)
	}
	//writeFn := fmt.Sprintf("%s/%s", ofp, objectPath)
	writeFn := makePath(ofp, objectPath)
	writer, err := os.OpenFile(writeFn, os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		log.Fatal("error opening writer file")
	}
	defer writer.Close()

	var signer *openpgp.Entity
	if signed {
		signer = entity
	} else {
		signer = nil
	}
	wc, err := openpgp.Encrypt(writer, entityList, signer, nil, config)
	if err != nil {
		log.Fatal("error getting writer closer during encryption")
	}
	defer wc.Close()

	_, err = wc.Write(br)
	if err != nil {
		log.Fatal("error writing input")
	}

	return digest, writeFn
}

func EncryptFile(ctx *LocalContext, fn, ofp string, config *packet.Config) (string, string) {
	var entity *openpgp.Entity
	signed := false
	if _, ok := (ctx.pgp).(PrivatePgpInfo); ok {
		// get private entity
		entity = getEntityById(getPrivKeyringDir(), ctx.key())
		if entity == nil {
			log.Fatal("key not found")
		}
		passphrase := ctx.pass()
		err := entity.PrivateKey.Decrypt(passphrase)
		if err != nil {
			log.Fatal("error decrypting private key")
		}
		signed = true
	} else {
		entity = getEntityById(getPubKeyringDir(), ctx.key())
	}
	return encryptFileHelper(fn, ofp, entity, signed, config)
}

// encrypt, and sign a file and output it to a new file with extension pgp
func signHelper(fn, keyId string, passphrase []byte) {
	entityList := getEntityList(getPrivKeyringDir())
	entity := filterEntityById(keyId, &entityList)
	// key
	err := entity.PrivateKey.Decrypt(passphrase)
	if err != nil {
		log.Fatal("error decrypt private key by using passphrase")
	}
	file, err := os.Open(fn)
	if err != nil {
		log.Fatal("error opening input file")
	}
	defer file.Close()
	writer, err := os.OpenFile(fn+".signed", os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		log.Fatal("error creating writer")
	}
	defer writer.Close()
	defer file.Close()
	err = openpgp.ArmoredDetachSign(file, entity, file, nil)
	if err != nil {
		log.Fatal("error signing")
	}
}

func sign(fn, keyId string) {
	// get passphrase
	passphrase := getPassphraseFromStdin()
	signHelper(fn, keyId, passphrase)
}

type PgpMismatchError struct{}

func (e *PgpMismatchError) Error() string {
	return "PublicPgpInfo cannot be used to generate prompt"
}

func DecryptFile(fn string, config *packet.Config, prompt openpgp.PromptFunction) string {
	input, err := os.Open(fn)
	defer input.Close()
	if err != nil {
		log.Fatal("error opening input file during decrytion")
	}
	entityList := getEntityList(getPrivKeyringDir())

	writeFn := fn + ".decrypt"
	writer, err := os.OpenFile(writeFn, os.O_WRONLY|os.O_CREATE, 0664)
	if err != nil {
		log.Fatal("error opening input file")
	}
	defer writer.Close()

	md, err := openpgp.ReadMessage(input, entityList, prompt, config)
	if err != nil {
		log.Fatal("error reading message: ", err.Error())
	}
	byteArray, err := ioutil.ReadAll(md.UnverifiedBody)
	_, err = writer.Write(byteArray)
	if err != nil {
		log.Fatal("error writiting to output file")
	}

	return writeFn
}

type SigInfo struct {
	SignedByKeyId uint64
	SignedBy      *openpgp.Key
}

func GetSignature(fn string, config *packet.Config, prompt openpgp.PromptFunction) (SigInfo, bool) {
	input, err := os.Open(fn)
	defer input.Close()
	if err != nil {
		log.Fatal("error opening input file during decrytion")
	}

	entityList := getEntityList(getPrivKeyringDir())

	md, err := openpgp.ReadMessage(input, entityList, prompt, config)
	if err != nil {
		log.Fatal("error reading message from input file")
	}
	if md.IsSigned {
		return SigInfo{
			SignedByKeyId: md.SignedByKeyId,
			SignedBy:      md.SignedBy,
		}, true
	}
	return SigInfo{}, false
}
