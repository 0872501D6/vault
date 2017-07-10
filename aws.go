package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
)

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/glacier"
)

// Return the tree hash for a given file
// Return error if any occurs
// The tree hash is computed by the aws supplied function
func TreeHash(body io.ReadSeeker) string {
	return hex.EncodeToString(glacier.ComputeHashes(body).TreeHash)
}

// Return a new Glacier service
func NewService(region string) *glacier.Glacier {
	sess := session.Must(session.NewSession(
		&aws.Config{
			Region: aws.String(region),
		}))
	service := glacier.New(sess)
	return service
}

// Upload a file for a given file name
// Return error if any occurs
func UploadFile(fn, vault string, service *glacier.Glacier) (*glacier.ArchiveCreationOutput, error) {
	fileBytes, err := ioutil.ReadFile(fn)
	if err != nil {
		log.Fatal(err.Error())
	}
	body := io.ReadSeeker(bytes.NewReader(fileBytes))
	digest := aws.String(TreeHash(body))
	// prepare upload input
	input := &glacier.UploadArchiveInput{
		AccountId:          aws.String("-"),
		ArchiveDescription: aws.String(fn),
		Body:               body,
		Checksum:           digest,
		VaultName:          aws.String(vault),
	}
	// upload archive
	resp, err := service.UploadArchive(input)
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(resp)
	return resp, nil
}
