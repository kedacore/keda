package kafka

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/segmentio/kafka-go/protocol/alteruserscramcredentials"
)

// AlterUserScramCredentialsRequest represents a request sent to a kafka broker to
// alter user scram credentials.
type AlterUserScramCredentialsRequest struct {
	// Address of the kafka broker to send the request to.
	Addr net.Addr

	// List of credentials to delete.
	Deletions []UserScramCredentialsDeletion

	// List of credentials to upsert.
	Upsertions []UserScramCredentialsUpsertion
}

type ScramMechanism int8

const (
	ScramMechanismUnknown ScramMechanism = iota // 0
	ScramMechanismSha256                        // 1
	ScramMechanismSha512                        // 2
)

type UserScramCredentialsDeletion struct {
	Name      string
	Mechanism ScramMechanism
}

type UserScramCredentialsUpsertion struct {
	Name           string
	Mechanism      ScramMechanism
	Iterations     int
	Salt           []byte
	SaltedPassword []byte
}

// AlterUserScramCredentialsResponse represents a response from a kafka broker to an alter user
// credentials request.
type AlterUserScramCredentialsResponse struct {
	// The amount of time that the broker throttled the request.
	Throttle time.Duration

	// List of altered user scram credentials.
	Results []AlterUserScramCredentialsResponseUser
}

type AlterUserScramCredentialsResponseUser struct {
	User  string
	Error error
}

// AlterUserScramCredentials sends user scram credentials alteration request to a kafka broker and returns
// the response.
func (c *Client) AlterUserScramCredentials(ctx context.Context, req *AlterUserScramCredentialsRequest) (*AlterUserScramCredentialsResponse, error) {
	deletions := make([]alteruserscramcredentials.RequestUserScramCredentialsDeletion, len(req.Deletions))
	upsertions := make([]alteruserscramcredentials.RequestUserScramCredentialsUpsertion, len(req.Upsertions))

	for deletionIdx, deletion := range req.Deletions {
		deletions[deletionIdx] = alteruserscramcredentials.RequestUserScramCredentialsDeletion{
			Name:      deletion.Name,
			Mechanism: int8(deletion.Mechanism),
		}
	}

	for upsertionIdx, upsertion := range req.Upsertions {
		upsertions[upsertionIdx] = alteruserscramcredentials.RequestUserScramCredentialsUpsertion{
			Name:           upsertion.Name,
			Mechanism:      int8(upsertion.Mechanism),
			Iterations:     int32(upsertion.Iterations),
			Salt:           upsertion.Salt,
			SaltedPassword: upsertion.SaltedPassword,
		}
	}

	m, err := c.roundTrip(ctx, req.Addr, &alteruserscramcredentials.Request{
		Deletions:  deletions,
		Upsertions: upsertions,
	})
	if err != nil {
		return nil, fmt.Errorf("kafka.(*Client).AlterUserScramCredentials: %w", err)
	}

	res := m.(*alteruserscramcredentials.Response)
	responseEntries := make([]AlterUserScramCredentialsResponseUser, len(res.Results))

	for responseIdx, responseResult := range res.Results {
		responseEntries[responseIdx] = AlterUserScramCredentialsResponseUser{
			User:  responseResult.User,
			Error: makeError(responseResult.ErrorCode, responseResult.ErrorMessage),
		}
	}
	ret := &AlterUserScramCredentialsResponse{
		Throttle: makeDuration(res.ThrottleTimeMs),
		Results:  responseEntries,
	}

	return ret, nil
}
