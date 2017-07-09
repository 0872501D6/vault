## Objective
Vault is a command line tool that encrypts and transfers the local files to aws 
data storage (Glacier or S3).

## Features
0. Configuration
  A new vault can be initialised by invoking
  ```
  vault init
  ```

  It will initialise a `.vault` config folder with the structure similar as
  follows:
  ```
  .vault
  ├── cache
  ├── config
  ├── credentials
  └── db
      ├── 000000.vlog
          ├── 000001.sst
              └── clog
  ```

  * `cache` stores the encrypted cached objects created by `add` command
  * `config` stores the key value configuration pairs
  * `credentials` stores the AWS access credentials
  * `db` is initialised by `badger` key value store

  Examples:
  ```
  vault config signingkey=PGP signing key
  vault config key=AWS access key ID
  vault config secret=AWS secret access key
  vault config region=AWS service region
  ```
The configuration is written as a config file, which will be read each time the
following commands are invoked. The format is key-value configuration, separated
by a single `=` assignment operator.

1. Add 
  ```
  vault add FILE_NAME/PATH_NAME
  ```

  `add` command adds files, or folders and encrypts them. The encrypted data are
  saved into `cache` folder. The cached data are managed by a local data store.
  Initially, the key value data store looks like this:

  ```go
  type VaultKey string // raw hash value

  type VaultFile struct {
      Hash    string   `json:hash`    // raw hash value computed by glacier SHA256 tree hasher
      Aliases []string `json:aliases` // all file path relevant to the vault config path
      Glacier string   `json:glacier` // glacier id
      KeyId   string   `json:keyid`   // openpgp key id, last 32 bit in hex
  }
  ```
  Files are identified by its tree hash value. Paths are unreliable in this
  context, as the content could be changed. There could be duplicates in the
  folder, and we only back it up once.

  - A prompt will be shown, asking for the password for encryption operation
  - If private key is not found by the program, fatal error occurs
 
2. Push
  ```
  vault push
  ```

  `push` command pushes the cached files to the remote. It is a synchornised
  operation. It will not terminate until all the uploads are finished. Each
  time when a response is received, the data store will be updated as well.

3. Update
  ```
  vault update
  ```

  `update` command updates the local data store about the remote data.

4. List
  ```
  vault list
  ```

  `list` command lists all the files that are backed up on the server,
  regardless it's been cached locally or not.

5. Fetch
  ```
  vault fetch FILE_NAME
  ```

  After being invoked, it would first look for the URL or glacier ID based on
  the file name, and then fetch it from remote. If there is a local file with
  the same presented, then we have two situations.
  - If the local file is identical to the remote one, do nothing
  - If the local file is different from the remote one, promot message, and let
    user decide

  Two additional flags can be added: `--local` and `--remote`. If flagged as 
  `--local`, then local one will be preserved; if flagged with `--remote`, the
  remote file from server will be preseved.
