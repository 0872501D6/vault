package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

const (
	CONF_DIR = ".vault"
	CRED     = "credentials"
	CONFIG   = "config"
	CACHE    = "cache"
	DB       = "db"
)

func makePath(args ...string) string {
	return strings.Join(args, "/")
}

// Create an empty file to write with file permission 0664
func createEmptyFile(fn string) *os.File {
	f, err := os.OpenFile(fn, os.O_WRONLY|os.O_RDONLY|os.O_CREATE, 0664)
	if err != nil {
		log.Fatal(err.Error())
	}
	return f
}

// Create an empty directory with file permission 0755
func createEmptyDir(dir string) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		os.Mkdir(dir, 0755)
	}
}

// Determines if the path exists or not
func dirExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// Determines if the current directory is where .vault config folder resides
func isCurrentVault() bool {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err.Error())
	}
	confDir := fmt.Sprintf("%s/%s", cwd, CONF_DIR)
	_, err = os.Stat(confDir)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// Starts from the current directory, and going up one by one
// terminates if there is no vault even at the root /
func governedByVault() (string, bool) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal("cannot get current working directory")
		return "", false
	}
	pathArray := strings.Split(cwd, "/")
	return recursiveDirExists(pathArray)
}

// Initialise a Config folder with an default config file
func InitConfig() {
	// create a new hidden folder for config
	path := fmt.Sprintf("%s", CONF_DIR)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, 0775)
		// get current working directory
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("cannot get current working directory")
		}
		fmt.Printf("Initialised a new little vault in %s/%s\n", wd, CONF_DIR)
	} else {
		log.Fatal("Vault config already exists for the current directory")
	}
	// create config file
	conf := fmt.Sprintf("%s/%s", path, CONFIG)
	confFile := createEmptyFile(conf)
	defer confFile.Close()
	// create a credential file
	cred := fmt.Sprintf("%s/%s", path, CRED)
	credFile := createEmptyFile(cred)
	credFile.Close()
	// initialise embedded database
	db := fmt.Sprintf("%s/%s", path, DB)
	createEmptyDir(db)
	kv := LoadBadger(db)
	defer kv.Close()
	// create an empty cache dir
	cache := fmt.Sprintf("%s/%s", path, CACHE)
	createEmptyDir(cache)
}

func OpenFile(fn string) *os.File {
	file, err := os.Open(fn)
	if err != nil {
		log.Fatal("Cannot open file ", fn)
	}
	return file
}

// Use simplified config key name
func isCredConfig(key string) bool {
	//return key == "aws_access_key_id" || key == "aws_secret_access_key"
	return key == "key" || key == "secret"
}

// Read config path and return as a key value map
func ReadConfig(path string) map[string]string {
	configs := make(map[string]string)
	file := OpenFile(path)
	defer file.Close()
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal("error on reading ", err.Error())
		}
		tokens := strings.Split(line, "=")
		if len(tokens) != 2 {
			continue
		}
		key, value := tokens[0], strings.TrimSuffix(tokens[1], "\n")
		configs[key] = value
	}
	return configs
}

// Writes the configuration map back to the file
func WriteConfig(path string, configs map[string]string) {
	file := createEmptyFile(path)
	defer file.Close()
	for k, v := range configs {
		line := k + "=" + v + "\n"
		file.WriteString(line)
	}
}

// SetConfig updates the old configuration from the values from argument
// It is the minimal implementation as it removes all the comments from the
// configuration as well
// TOML will probably be more suitable in the future
func SetConfig(fs *flag.FlagSet) {
	if vaultDirPath, b := governedByVault(); b {
		// update the config data
		confPath := fmt.Sprintf("%s/%s/%s", vaultDirPath, CONF_DIR, CONFIG)
		conf := ReadConfig(confPath)
		// update the cred data
		credPath := fmt.Sprintf("%s/%s/%s", vaultDirPath, CONF_DIR, CRED)
		cred := ReadConfig(credPath)

		// update or insert key value map
		for _, pair := range fs.Args() {
			tokens := strings.Split(pair, "=")
			if len(tokens) != 2 {
				continue
			}
			// some conversions happens over here
			key, val := tokens[0], tokens[1]
			if isCredConfig(key) {
				switch key {
				case "key":
					key = "aws_access_key_id"
				case "secret":
					key = "aws_secret_access_key"
				}
				cred[key] = val
			} else {
				conf[key] = val
			}
		}
		// delete old config file
		// TODO (archfiery) when the amount of config is not huge, we do this
		// rewrite
		os.Remove(confPath)
		os.Remove(credPath)

		// create a new one with updated conf
		WriteConfig(confPath, conf)
		WriteConfig(credPath, cred)
	} else {
		log.Fatal("Vault uninitialised")
	}
}

type FlagWrap struct {
	Name    string
	FlagSet *flag.FlagSet
}

// config command flag set
func configFlagSet() FlagWrap {
	// by default, these config values work after we set AWS_SDK_LOAD_CONFIG=1
	flagSet := flag.NewFlagSet("config", flag.ExitOnError)
	flagSet.String("key", "", "AWS access key ID")
	flagSet.String("secret", "", "AWS secret access key")
	flagSet.String("region", "", "AWS service region")
	flagSet.String("signingkey", "", "Your PGP signing key")
	return FlagWrap{"config", flagSet}
}

// init command flag set
func initFlagSet() FlagWrap {
	initSet := flag.NewFlagSet("init", flag.ExitOnError)
	return FlagWrap{"init", initSet}
}

// add command flag set
func addFlagSet() FlagWrap {
	addSet := flag.NewFlagSet("add", flag.ExitOnError)
	return FlagWrap{"add", addSet}
}

// set aws environment variables
func setAwsEnv(vaultDir string) {
	os.Setenv("AWS_SDK_LOAD_CONFIG", "1")
	credentialPath := fmt.Sprintf("%s/%s/credentials", vaultDir, CONF_DIR)
	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credentialPath)
}

func printDefaults(fw []FlagWrap) {
	for _, f := range fw {
		fmt.Printf("vault %s\n", f.Name)
		f.FlagSet.PrintDefaults()
	}
}

func main() {
	// parse flag set
	initCommand := initFlagSet()
	configCommand := configFlagSet()
	addCommand := addFlagSet()
	flags := []FlagWrap{initCommand, configCommand, addCommand}

	if len(os.Args) < 2 {
		fmt.Println("Please specify an action")
		printDefaults(flags)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		initCommand.FlagSet.Parse(os.Args[2:])
	case "config":
		configCommand.FlagSet.Parse(os.Args[2:])
	case "add":
		addCommand.FlagSet.Parse(os.Args[2:])
	default:
		printDefaults(flags)
		os.Exit(1)
	}

	vaultDir, isGoverned := governedByVault()
	setAwsEnv(vaultDir)
	if !isGoverned {
		// parse commands
		if initCommand.FlagSet.Parsed() {
			InitConfig()
		} else {
			log.Fatal("Vault uninitialised")
		}
	} else {
		// parse commands
		if initCommand.FlagSet.Parsed() {
			if !isCurrentVault() {
				InitConfig()
			} else {
				log.Fatal("Vault already initialised. Exit")
			}
		} else if configCommand.FlagSet.Parsed() {
			SetConfig(configCommand.FlagSet)
		} else if addCommand.FlagSet.Parsed() {
			ctx := NewLocalContext(true, getPassphraseFromStdin)
			AddCache(&ctx, os.Args[2:])
		}
	}
}
