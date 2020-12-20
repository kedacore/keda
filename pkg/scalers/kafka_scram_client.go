package scalers

import (
	"crypto/sha256"
	"crypto/sha512"

	"github.com/xdg/scram"
)

// SHA256 hash generator function for SCRAM conversation
var SHA256 scram.HashGeneratorFcn = sha256.New

// SHA512 hash generator function for SCRAM conversation
var SHA512 scram.HashGeneratorFcn = sha512.New

// XDGSCRAMClient struct to perform SCRAM conversation
type XDGSCRAMClient struct {
	*scram.Client
	*scram.ClientConversation
	scram.HashGeneratorFcn
}

// Begin starts SCRAM conversation
func (x *XDGSCRAMClient) Begin(userName, password, authzID string) (err error) {
	x.Client, err = x.HashGeneratorFcn.NewClient(userName, password, authzID)
	if err != nil {
		return err
	}
	x.ClientConversation = x.Client.NewConversation()
	return nil
}

// Step performs step in SCRAM conversation
func (x *XDGSCRAMClient) Step(challenge string) (response string, err error) {
	response, err = x.ClientConversation.Step(challenge)
	return
}

// Done completes SCRAM conversation
func (x *XDGSCRAMClient) Done() bool {
	return x.ClientConversation.Done()
}
