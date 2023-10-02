package kafka

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/segmentio/kafka-go/protocol/describeuserscramcredentials"
)

// DescribeUserScramCredentialsRequest represents a request sent to a kafka broker to
// describe user scram credentials.
type DescribeUserScramCredentialsRequest struct {
	// Address of the kafka broker to send the request to.
	Addr net.Addr

	// List of Scram users to describe
	Users []UserScramCredentialsUser
}

type UserScramCredentialsUser struct {
	Name string
}

// DescribeUserScramCredentialsResponse represents a response from a kafka broker to a describe user
// credentials request.
type DescribeUserScramCredentialsResponse struct {
	// The amount of time that the broker throttled the request.
	Throttle time.Duration

	// Top level error that occurred while attempting to describe
	// the user scram credentials.
	//
	// The errors contain the kafka error code. Programs may use the standard
	// errors.Is function to test the error against kafka error codes.
	Error error

	// List of described user scram credentials.
	Results []DescribeUserScramCredentialsResponseResult
}

type DescribeUserScramCredentialsResponseResult struct {
	User            string
	CredentialInfos []DescribeUserScramCredentialsCredentialInfo
	Error           error
}

type DescribeUserScramCredentialsCredentialInfo struct {
	Mechanism  ScramMechanism
	Iterations int
}

// DescribeUserScramCredentials sends a user scram credentials describe request to a kafka broker and returns
// the response.
func (c *Client) DescribeUserScramCredentials(ctx context.Context, req *DescribeUserScramCredentialsRequest) (*DescribeUserScramCredentialsResponse, error) {
	users := make([]describeuserscramcredentials.RequestUser, len(req.Users))

	for userIdx, user := range req.Users {
		users[userIdx] = describeuserscramcredentials.RequestUser{
			Name: user.Name,
		}
	}

	m, err := c.roundTrip(ctx, req.Addr, &describeuserscramcredentials.Request{
		Users: users,
	})
	if err != nil {
		return nil, fmt.Errorf("kafka.(*Client).DescribeUserScramCredentials: %w", err)
	}

	res := m.(*describeuserscramcredentials.Response)
	responseResults := make([]DescribeUserScramCredentialsResponseResult, len(res.Results))

	for responseIdx, responseResult := range res.Results {
		credentialInfos := make([]DescribeUserScramCredentialsCredentialInfo, len(responseResult.CredentialInfos))

		for credentialInfoIdx, credentialInfo := range responseResult.CredentialInfos {
			credentialInfos[credentialInfoIdx] = DescribeUserScramCredentialsCredentialInfo{
				Mechanism:  ScramMechanism(credentialInfo.Mechanism),
				Iterations: int(credentialInfo.Iterations),
			}
		}
		responseResults[responseIdx] = DescribeUserScramCredentialsResponseResult{
			User:            responseResult.User,
			CredentialInfos: credentialInfos,
			Error:           makeError(responseResult.ErrorCode, responseResult.ErrorMessage),
		}
	}
	ret := &DescribeUserScramCredentialsResponse{
		Throttle: makeDuration(res.ThrottleTimeMs),
		Error:    makeError(res.ErrorCode, res.ErrorMessage),
		Results:  responseResults,
	}

	return ret, nil
}
