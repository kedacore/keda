// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package internal

import (
	"context"

	azlog "github.com/Azure/azure-sdk-for-go/sdk/internal/log"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/amqpwrap"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/auth"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/exported"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/go-amqp"
)

const (
	cbsAddress           = "$cbs"
	cbsOperationKey      = "operation"
	cbsOperationPutToken = "put-token"
	cbsTokenTypeKey      = "type"
	cbsAudienceKey       = "name"
	cbsExpirationKey     = "expiration"
)

// NegotiateClaim attempts to put a token to the $cbs management endpoint to negotiate auth for the given audience
func NegotiateClaim(ctx context.Context, audience string, conn amqpwrap.AMQPClient, provider auth.TokenProvider) error {
	link, err := NewRPCLink(ctx, RPCLinkArgs{
		Client:   conn,
		Address:  cbsAddress,
		LogEvent: exported.EventAuth,
	})

	if err != nil {
		return err
	}

	defer func() {
		if err := link.Close(ctx); err != nil {
			azlog.Writef(exported.EventAuth, "Failed closing claim link: %s", err.Error())
		}
	}()

	token, err := provider.GetToken(audience)
	if err != nil {
		return err
	}

	azlog.Writef(exported.EventAuth, "negotiating claim for audience %s with token type %s and expiry of %s", audience, token.TokenType, token.Expiry)

	msg := &amqp.Message{
		Value: token.Token,
		ApplicationProperties: map[string]interface{}{
			cbsOperationKey:  cbsOperationPutToken,
			cbsTokenTypeKey:  string(token.TokenType),
			cbsAudienceKey:   audience,
			cbsExpirationKey: token.Expiry,
		},
	}

	if _, err := link.RPC(ctx, msg); err != nil {
		return err
	}

	return nil
}
