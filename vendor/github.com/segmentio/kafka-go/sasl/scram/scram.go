package scram

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"github.com/segmentio/kafka-go/sasl"
	"github.com/xdg-go/scram"
)

// Algorithm determines the hash function used by SCRAM to protect the user's
// credentials.
type Algorithm interface {
	// Name returns the algorithm's name, e.g. "SCRAM-SHA-256"
	Name() string

	// Hash returns a new hash.Hash.
	Hash() hash.Hash
}

type sha256Algo struct{}

func (sha256Algo) Name() string {
	return "SCRAM-SHA-256"
}

func (sha256Algo) Hash() hash.Hash {
	return sha256.New()
}

type sha512Algo struct{}

func (sha512Algo) Name() string {
	return "SCRAM-SHA-512"
}

func (sha512Algo) Hash() hash.Hash {
	return sha512.New()
}

var (
	SHA256 Algorithm = sha256Algo{}
	SHA512 Algorithm = sha512Algo{}
)

type mechanism struct {
	algo   Algorithm
	client *scram.Client
}

type session struct {
	convo *scram.ClientConversation
}

// Mechanism returns a new sasl.Mechanism that will use SCRAM with the provided
// Algorithm to securely transmit the provided credentials to Kafka.
//
// SCRAM-SHA-256 and SCRAM-SHA-512 were added to Kafka in 0.10.2.0.  These
// mechanisms will not work with older versions.
func Mechanism(algo Algorithm, username, password string) (sasl.Mechanism, error) {
	hashGen := scram.HashGeneratorFcn(algo.Hash)
	client, err := hashGen.NewClient(username, password, "")
	if err != nil {
		return nil, err
	}

	return &mechanism{
		algo:   algo,
		client: client,
	}, nil
}

func (m *mechanism) Name() string {
	return m.algo.Name()
}

func (m *mechanism) Start(ctx context.Context) (sasl.StateMachine, []byte, error) {
	convo := m.client.NewConversation()
	str, err := convo.Step("")
	if err != nil {
		return nil, nil, err
	}
	return &session{convo: convo}, []byte(str), nil
}

func (s *session) Next(ctx context.Context, challenge []byte) (bool, []byte, error) {
	str, err := s.convo.Step(string(challenge))
	return s.convo.Done(), []byte(str), err
}
