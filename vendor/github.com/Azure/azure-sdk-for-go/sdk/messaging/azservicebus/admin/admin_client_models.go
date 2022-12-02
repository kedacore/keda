// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package admin

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/atom"
)

// EntityStatus represents the current status of the entity.
type EntityStatus string

const (
	// EntityStatusActive indicates an entity can be used for sending and receiving.
	EntityStatusActive EntityStatus = "Active"
	// EntityStatusDisabled indicates an entity cannot be used for sending or receiving.
	EntityStatusDisabled EntityStatus = "Disabled"
	// EntityStatusSendDisabled indicates that an entity cannot be used for sending.
	EntityStatusSendDisabled EntityStatus = "SendDisabled"
	// EntityStatusReceiveDisabled indicates that an entity cannot be used for receiving.
	EntityStatusReceiveDisabled EntityStatus = "ReceiveDisabled"
)

type (
	// AccessRight is an access right (Manage, Send, Listen) for an AuthorizationRule.
	AccessRight string

	// AuthorizationRule is a rule with keys and rights associated with an entity.
	AuthorizationRule struct {
		// AccessRights for this rule.
		AccessRights []AccessRight

		// KeyName for this rule.
		KeyName *string

		// CreatedTime for this rule.
		CreatedTime *time.Time

		// ModifiedTime for this rule.
		ModifiedTime *time.Time

		// PrimaryKey for this rule.
		PrimaryKey *string

		// SecondaryKey for this rule.
		SecondaryKey *string
	}
)

const (
	// AccessRightManage allows changes to an entity.
	AccessRightManage AccessRight = "Manage"
	// AccessRightSend allows you to send messages to this entity.
	AccessRightSend AccessRight = "Send"
	// AccessRightListen allows you to receive messages from this entity.
	AccessRightListen AccessRight = "Listen"
)

func internalAccessRightsToPublic(internalRules []atom.AuthorizationRule) []AuthorizationRule {
	var rules []AuthorizationRule

	for _, rule := range internalRules {
		var accessRights []AccessRight

		for _, right := range rule.Rights {
			accessRights = append(accessRights, AccessRight(right))
		}

		rules = append(rules, AuthorizationRule{
			AccessRights: accessRights,
			KeyName:      rule.KeyName,
			CreatedTime:  rule.CreatedTime,
			ModifiedTime: rule.ModifiedTime,
			PrimaryKey:   rule.PrimaryKey,
			SecondaryKey: rule.SecondaryKey,
		})
	}

	return rules
}

func publicAccessRightsToInternal(rules []AuthorizationRule) []atom.AuthorizationRule {
	var internalRules []atom.AuthorizationRule

	for _, rule := range rules {
		var accessRights []string

		for _, right := range rule.AccessRights {
			accessRights = append(accessRights, string(right))
		}

		internalRules = append(internalRules, atom.AuthorizationRule{
			Type:         "SharedAccessAuthorizationRule",
			ClaimType:    "SharedAccessKey",
			ClaimValue:   "None",
			Rights:       accessRights,
			KeyName:      rule.KeyName,
			CreatedTime:  rule.CreatedTime,
			ModifiedTime: rule.ModifiedTime,
			PrimaryKey:   rule.PrimaryKey,
			SecondaryKey: rule.SecondaryKey,
		})
	}

	return internalRules
}
