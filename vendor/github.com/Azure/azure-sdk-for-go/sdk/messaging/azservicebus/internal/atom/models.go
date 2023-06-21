// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package atom

import (
	"encoding/xml"
	"time"
)

// All
type (
	AuthorizationRule struct {
		// Type is the type attribute, which indicates the type of AuthorizationRule
		// (today this is only `SharedAccessAuthorizationRule`)
		Type string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`

		ClaimType  string `xml:"ClaimType"`
		ClaimValue string `xml:"ClaimValue"`

		// SharedAccessAuthorizationRule properties
		Rights       []string   `xml:"Rights>AccessRights"`
		KeyName      *string    `xml:"KeyName"`
		CreatedTime  *time.Time `xml:"CreatedTime"`
		ModifiedTime *time.Time `xml:"ModifiedTime"`

		PrimaryKey   *string `xml:"PrimaryKey"`
		SecondaryKey *string `xml:"SecondaryKey"`
	}
)

// Queues
type (
	// QueueEntity is the Azure Service Bus description of a Queue for management activities
	QueueEntity struct {
		*QueueDescription
		*Entity
	}

	// QueueFeed is a specialized feed containing QueueEntries
	QueueFeed struct {
		*Feed
		Entries []QueueEnvelope `xml:"entry"`
	}

	// QueueEnvelope is a specialized Queue feed entry
	QueueEnvelope struct {
		*Entry
		Content *QueueContent `xml:"content"`
	}

	// QueueContent is a specialized Queue body for an Atom entry
	QueueContent struct {
		XMLName          xml.Name         `xml:"content"`
		Type             string           `xml:"type,attr"`
		QueueDescription QueueDescription `xml:"QueueDescription"`
	}

	// QueueDescription is the content type for Queue management requests
	QueueDescription struct {
		XMLName xml.Name `xml:"QueueDescription"`
		BaseEntityDescription
		LockDuration                        *string             `xml:"LockDuration,omitempty"`               // LockDuration - ISO 8601 timespan duration of a peek-lock; that is, the amount of time that the message is locked for other receivers. The maximum value for LockDuration is 5 minutes; the default value is 1 minute.
		MaxSizeInMegabytes                  *int32              `xml:"MaxSizeInMegabytes,omitempty"`         // MaxSizeInMegabytes - The maximum size of the queue in megabytes, which is the size of memory allocated for the queue. Default is 1024.
		RequiresDuplicateDetection          *bool               `xml:"RequiresDuplicateDetection,omitempty"` // RequiresDuplicateDetection - A value indicating if this queue requires duplicate detection.
		RequiresSession                     *bool               `xml:"RequiresSession,omitempty"`
		DefaultMessageTimeToLive            *string             `xml:"DefaultMessageTimeToLive,omitempty"`            // DefaultMessageTimeToLive - ISO 8601 default message timespan to live value. This is the duration after which the message expires, starting from when the message is sent to Service Bus. This is the default value used when TimeToLive is not set on a message itself.
		DeadLetteringOnMessageExpiration    *bool               `xml:"DeadLetteringOnMessageExpiration,omitempty"`    // DeadLetteringOnMessageExpiration - A value that indicates whether this queue has dead letter support when a message expires.
		DuplicateDetectionHistoryTimeWindow *string             `xml:"DuplicateDetectionHistoryTimeWindow,omitempty"` // DuplicateDetectionHistoryTimeWindow - ISO 8601 timeSpan structure that defines the duration of the duplicate detection history. The default value is 10 minutes.
		MaxDeliveryCount                    *int32              `xml:"MaxDeliveryCount,omitempty"`                    // MaxDeliveryCount - The maximum delivery count. A message is automatically deadlettered after this number of deliveries. default value is 10.
		EnableBatchedOperations             *bool               `xml:"EnableBatchedOperations,omitempty"`             // EnableBatchedOperations - Value that indicates whether server-side batched operations are enabled.
		SizeInBytes                         *int64              `xml:"SizeInBytes,omitempty"`                         // SizeInBytes - The size of the queue, in bytes.
		MessageCount                        *int64              `xml:"MessageCount,omitempty"`                        // MessageCount - The number of messages in the queue.
		IsAnonymousAccessible               *bool               `xml:"IsAnonymousAccessible,omitempty"`
		AuthorizationRules                  []AuthorizationRule `xml:"AuthorizationRules>AuthorizationRule,omitempty"`
		Status                              *EntityStatus       `xml:"Status,omitempty"`
		AccessedAt                          string              `xml:"AccessedAt,omitempty"`
		CreatedAt                           string              `xml:"CreatedAt,omitempty"`
		UpdatedAt                           string              `xml:"UpdatedAt,omitempty"`
		SupportOrdering                     *bool               `xml:"SupportOrdering,omitempty"`
		AutoDeleteOnIdle                    *string             `xml:"AutoDeleteOnIdle,omitempty"`
		EnablePartitioning                  *bool               `xml:"EnablePartitioning,omitempty"`
		EnableExpress                       *bool               `xml:"EnableExpress,omitempty"`
		CountDetails                        *CountDetails       `xml:"CountDetails,omitempty"`
		ForwardTo                           *string             `xml:"ForwardTo,omitempty"`
		ForwardDeadLetteredMessagesTo       *string             `xml:"ForwardDeadLetteredMessagesTo,omitempty"` // ForwardDeadLetteredMessagesTo - absolute URI of the entity to forward dead letter messages
		UserMetadata                        *string             `xml:"UserMetadata,omitempty"`
		MaxMessageSizeInKilobytes           *int64              `xml:"MaxMessageSizeInKilobytes,omitempty"`
	}
)

func (qf QueueFeed) Items() []QueueEnvelope {
	return qf.Entries
}

// Topics
type (
	// TopicEntity is the Azure Service Bus description of a Topic for management activities
	TopicEntity struct {
		*TopicDescription
		*Entity
	}

	// TopicEnvelope is a specialized Topic feed entry
	TopicEnvelope struct {
		*Entry
		Content *TopicContent `xml:"content"`
	}

	// TopicContent is a specialized Topic body for an Atom entry
	TopicContent struct {
		XMLName          xml.Name         `xml:"content"`
		Type             string           `xml:"type,attr"`
		TopicDescription TopicDescription `xml:"TopicDescription"`
	}

	// TopicFeed is a specialized feed containing Topic Entries
	TopicFeed struct {
		*Feed
		Entries []TopicEnvelope `xml:"entry"`
	}

	// TopicDescription is the content type for Topic management requests
	// Refer here for ordering constraints: https://github.com/Azure/azure-sdk-for-net/blob/ed2e86cb299e11a276dcf652a9db796efe2d2a27/sdk/servicebus/Azure.Messaging.ServiceBus/src/Administration/TopicPropertiesExtensions.cs#L178
	TopicDescription struct {
		XMLName xml.Name `xml:"TopicDescription"`
		BaseEntityDescription
		DefaultMessageTimeToLive            *string             `xml:"DefaultMessageTimeToLive,omitempty"`            // DefaultMessageTimeToLive - ISO 8601 default message time span to live value. This is the duration after which the message expires, starting from when the message is sent to Service Bus. This is the default value used when TimeToLive is not set on a message itself.
		MaxSizeInMegabytes                  *int32              `xml:"MaxSizeInMegabytes,omitempty"`                  // MaxSizeInMegabytes - The maximum size of the queue in megabytes, which is the size of memory allocated for the queue. Default is 1024.
		RequiresDuplicateDetection          *bool               `xml:"RequiresDuplicateDetection,omitempty"`          // RequiresDuplicateDetection - A value indicating if this queue requires duplicate detection.
		DuplicateDetectionHistoryTimeWindow *string             `xml:"DuplicateDetectionHistoryTimeWindow,omitempty"` // DuplicateDetectionHistoryTimeWindow - ISO 8601 timeSpan structure that defines the duration of the duplicate detection history. The default value is 10 minutes.
		EnableBatchedOperations             *bool               `xml:"EnableBatchedOperations,omitempty"`             // EnableBatchedOperations - Value that indicates whether server-side batched operations are enabled.
		SizeInBytes                         *int64              `xml:"SizeInBytes,omitempty"`                         // SizeInBytes - The size of the queue, in bytes.
		FilteringMessagesBeforePublishing   *bool               `xml:"FilteringMessagesBeforePublishing,omitempty"`
		IsAnonymousAccessible               *bool               `xml:"IsAnonymousAccessible,omitempty"`
		AuthorizationRules                  []AuthorizationRule `xml:"AuthorizationRules>AuthorizationRule,omitempty"`
		Status                              *EntityStatus       `xml:"Status,omitempty"`
		UserMetadata                        *string             `xml:"UserMetadata,omitempty"`
		AccessedAt                          string              `xml:"AccessedAt,omitempty"`
		CreatedAt                           string              `xml:"CreatedAt,omitempty"`
		UpdatedAt                           string              `xml:"UpdatedAt,omitempty"`
		SupportOrdering                     *bool               `xml:"SupportOrdering,omitempty"`
		AutoDeleteOnIdle                    *string             `xml:"AutoDeleteOnIdle,omitempty"`
		EnablePartitioning                  *bool               `xml:"EnablePartitioning,omitempty"`
		EnableSubscriptionPartitioning      *bool               `xml:"EnableSubscriptionPartitioning,omitempty"`
		EnableExpress                       *bool               `xml:"EnableExpress,omitempty"`
		CountDetails                        *CountDetails       `xml:"CountDetails,omitempty"`
		SubscriptionCount                   *int32              `xml:"SubscriptionCount,omitempty"`
		MaxMessageSizeInKilobytes           *int64              `xml:"MaxMessageSizeInKilobytes,omitempty"`
	}
)

func (tf TopicFeed) Items() []TopicEnvelope {
	return tf.Entries
}

// Subscriptions (and rules)
type (
	// RuleDescription is the content type for Subscription Rule management requests
	RuleDescription struct {
		XMLName xml.Name `xml:"RuleDescription"`
		XMLNS   string   `xml:"xmlns,attr"`
		XMLNSI  string   `xml:"xmlns:i,attr"`
		BaseEntityDescription
		CreatedAt string             `xml:"CreatedAt,omitempty"`
		Filter    *FilterDescription `xml:"Filter,omitempty"`
		Action    *ActionDescription `xml:"Action,omitempty"`
		Name      string             `xml:"Name"`
	}
	// DefaultRuleDescription is the content type for Subscription Rule management requests
	DefaultRuleDescription struct {
		XMLName xml.Name           `xml:"DefaultRuleDescription"`
		Filter  *FilterDescription `xml:"Filter"`
		Action  *ActionDescription `xml:"Action,omitempty"`
		Name    string             `xml:"Name,omitempty"`
	}

	// FilterDescription describes a filter which can be applied to a subscription to filter messages from the topic.
	//
	// Subscribers can define which messages they want to receive from a topic. These messages are specified in the
	// form of one or more named subscription rules. Each rule consists of a condition that selects particular messages
	// and an action that annotates the selected message. For each matching rule condition, the subscription produces a
	// copy of the message, which may be differently annotated for each matching rule.
	//
	// Each newly created topic subscription has an initial default subscription rule. If you don't explicitly specify a
	// filter condition for the rule, the applied filter is the true filter that enables all messages to be selected
	// into the subscription. The default rule has no associated annotation action.
	FilterDescription struct {
		XMLName xml.Name `xml:"Filter"`

		// RawXML is any XML that wasn't covered by our known properties.
		RawXML []byte `xml:",innerxml"`

		// RawAttrs are attributes for the raw XML element that wasn't covered by our known properties.
		RawAttrs []xml.Attr `xml:",any,attr"`

		CorrelationFilter
		Type               string  `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
		SQLExpression      *string `xml:"SqlExpression,omitempty"`
		CompatibilityLevel int     `xml:"CompatibilityLevel,omitempty"`

		Parameters *KeyValueList `xml:"Parameters,omitempty"`
	}

	// ActionDescription describes an action upon a message that matches a filter
	//
	// With SQL filter conditions, you can define an action that can annotate the message by adding, removing, or
	// replacing properties and their values. The action uses a SQL-like expression that loosely leans on the SQL
	// UPDATE statement syntax. The action is performed on the message after it has been matched and before the message
	// is selected into the subscription. The changes to the message properties are private to the message copied into
	// the subscription.
	ActionDescription struct {
		Type          string        `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
		SQLExpression string        `xml:"SqlExpression,omitempty"`
		Parameters    *KeyValueList `xml:"Parameters,omitempty"`

		// RawXML is any XML that wasn't covered by our known properties.
		RawXML []byte `xml:",innerxml"`

		// RawAttrs are attributes for the raw XML element that wasn't covered by our known properties.
		RawAttrs []xml.Attr `xml:",any,attr"`
	}

	Value struct {
		Type  string `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
		L28NS string `xml:"xmlns:l28,attr"`
		Text  string `xml:",chardata"`
	}

	KeyValueOfstringanyType struct {
		Key   string `xml:"Key"`
		Value Value  `xml:"Value"`
	}

	// RuleEntity is the Azure Service Bus description of a Subscription Rule for madnagement activities
	RuleEntity struct {
		*RuleDescription
		*Entity
	}

	// RuleContent is a specialized Subscription body for an Atom entry
	RuleContent struct {
		XMLName         xml.Name        `xml:"content"`
		Type            string          `xml:"type,attr"`
		RuleDescription RuleDescription `xml:"RuleDescription"`
	}

	TempRuleEnvelope struct {
		*Entry
	}

	// RuleFeed is a specialized feed containing RuleEnvelopes
	RuleFeed struct {
		*Feed
		Entries []RuleEnvelope `xml:"entry"`
	}

	RuleEnvelope struct {
		*Entry
		Content *RuleContent `xml:"content"`
	}

	// SubscriptionDescription is the content type for Subscription management requests
	SubscriptionDescription struct {
		XMLName xml.Name `xml:"SubscriptionDescription"`
		BaseEntityDescription
		LockDuration                              *string                 `xml:"LockDuration,omitempty"` // LockDuration - ISO 8601 timespan duration of a peek-lock; that is, the amount of time that the message is locked for other receivers. The maximum value for LockDuration is 5 minutes; the default value is 1 minute.
		RequiresSession                           *bool                   `xml:"RequiresSession,omitempty"`
		DefaultMessageTimeToLive                  *string                 `xml:"DefaultMessageTimeToLive,omitempty"`         // DefaultMessageTimeToLive - ISO 8601 default message timespan to live value. This is the duration after which the message expires, starting from when the message is sent to Service Bus. This is the default value used when TimeToLive is not set on a message itself.
		DeadLetteringOnMessageExpiration          *bool                   `xml:"DeadLetteringOnMessageExpiration,omitempty"` // DeadLetteringOnMessageExpiration - A value that indicates whether this queue has dead letter support when a message expires.
		DeadLetteringOnFilterEvaluationExceptions *bool                   `xml:"DeadLetteringOnFilterEvaluationExceptions,omitempty"`
		DefaultRuleDescription                    *DefaultRuleDescription `xml:"DefaultRuleDescription,omitempty"`  // DefaultRuleDescription - A  default rule that is created right when the new subscription is created.
		MaxDeliveryCount                          *int32                  `xml:"MaxDeliveryCount,omitempty"`        // MaxDeliveryCount - The maximum delivery count. A message is automatically deadlettered after this number of deliveries. default value is 10.
		MessageCount                              *int64                  `xml:"MessageCount,omitempty"`            // MessageCount - The number of messages in the queue.
		EnableBatchedOperations                   *bool                   `xml:"EnableBatchedOperations,omitempty"` // EnableBatchedOperations - Value that indicates whether server-side batched operations are enabled.
		Status                                    *EntityStatus           `xml:"Status,omitempty"`
		ForwardTo                                 *string                 `xml:"ForwardTo,omitempty"` // ForwardTo - absolute URI of the entity to forward messages
		UserMetadata                              *string                 `xml:"UserMetadata,omitempty"`
		ForwardDeadLetteredMessagesTo             *string                 `xml:"ForwardDeadLetteredMessagesTo,omitempty"` // ForwardDeadLetteredMessagesTo - absolute URI of the entity to forward dead letter messages
		AutoDeleteOnIdle                          *string                 `xml:"AutoDeleteOnIdle,omitempty"`
		CreatedAt                                 string                  `xml:"CreatedAt,omitempty"`
		UpdatedAt                                 string                  `xml:"UpdatedAt,omitempty"`
		AccessedAt                                string                  `xml:"AccessedAt,omitempty"`
		CountDetails                              *CountDetails           `xml:"CountDetails,omitempty"`
	}

	// SubscriptionEntity is the Azure Service Bus description of a topic Subscription for management activities
	SubscriptionEntity struct {
		*SubscriptionDescription
		*Entity
	}

	// SubscriptionFeed is a specialized feed containing Topic Subscriptions
	SubscriptionFeed struct {
		*Feed
		Entries []SubscriptionEnvelope `xml:"entry"`
	}

	// subscriptionEntryContent is a specialized Topic feed Subscription
	SubscriptionEnvelope struct {
		*Entry
		Content *SubscriptionContent `xml:"content"`
	}

	// SubscriptionContent is a specialized Subscription body for an Atom entry
	SubscriptionContent struct {
		XMLName                 xml.Name                `xml:"content"`
		Type                    string                  `xml:"type,attr"`
		SubscriptionDescription SubscriptionDescription `xml:"SubscriptionDescription"`
	}

	// Entity is represents the most basic form of an Azure Service Bus entity.
	Entity struct {
		Name string
		ID   string
	}
)

func (sf SubscriptionFeed) Items() []SubscriptionEnvelope {
	return sf.Entries
}

func (rf RuleFeed) Items() []RuleEnvelope {
	return rf.Entries
}

// Filters
type (
	// CorrelationFilter holds a set of conditions that are matched against one or more of an arriving message's user
	// and system properties. A common use is to match against the CorrelationId property, but the application can also
	// choose to match against ContentType, Label, MessageId, ReplyTo, ReplyToSessionId, SessionId, To, and any
	// user-defined properties. A match exists when an arriving message's value for a property is equal to the value
	// specified in the correlation filter. For string expressions, the comparison is case-sensitive. When specifying
	// multiple match properties, the filter combines them as a logical AND condition, meaning for the filter to match,
	// all conditions must match.
	CorrelationFilter struct {
		CorrelationID    *string       `xml:"CorrelationId,omitempty"`
		MessageID        *string       `xml:"MessageId,omitempty"`
		To               *string       `xml:"To,omitempty"`
		ReplyTo          *string       `xml:"ReplyTo,omitempty"`
		Label            *string       `xml:"Label,omitempty"`
		SessionID        *string       `xml:"SessionId,omitempty"`
		ReplyToSessionID *string       `xml:"ReplyToSessionId,omitempty"`
		ContentType      *string       `xml:"ContentType,omitempty"`
		Properties       *KeyValueList `xml:"Properties,omitempty"`
	}

	KeyValueList struct {
		KeyValues []KeyValueOfstringanyType `xml:"KeyValueOfstringanyType,omitempty"`
	}
)

type (
	/*
		<entry xmlns="http://www.w3.org/2005/Atom">
			<id>https://<my servicebus name>.servicebus.windows.net/$namespaceinfo?api-version=2017-04</id>
			<title type="text"><my servicebus name></title>
			<updated>2021-11-07T23:41:24Z</updated>
			<author>
				<name><my servicebus name></name>
			</author>
			<link rel="self" href="https://<my servicebus name>.servicebus.windows.net/$namespaceinfo?api-version=2017-04"></link>
			<content type="application/xml">
				<NamespaceInfo xmlns="http://schemas.microsoft.com/netservices/2010/10/servicebus/connect" xmlns:i="http://www.w3.org/2001/XMLSchema-instance">
					<CreatedTime>2019-12-03T22:18:04.09Z</CreatedTime>
					<MessagingSKU>Standard</MessagingSKU>
					<ModifiedTime>2021-08-19T23:37:00.75Z</ModifiedTime>
					<Name><my servicebus name></Name>
					<NamespaceType>Messaging</NamespaceType>
				</NamespaceInfo>
			</content>
		</entry>
	*/

	NamespaceEntry struct {
		NamespaceInfo *NamespaceInfo `xml:"content>NamespaceInfo"`
	}

	NamespaceInfo struct {
		CreatedTime    string `xml:"CreatedTime"`
		MessagingSKU   string `xml:"MessagingSKU"`
		MessagingUnits *int64 `xml:"MessagingUnits"`
		ModifiedTime   string `xml:"ModifiedTime"`
		Name           string `xml:"Name"`
	}
)

func StringToTime(timeStr string) (time.Time, error) {
	// The ATOM API can return `0001-01-01T00:00:00` as the 'zero' value for the AccessedAt
	// value when you first create an entity. In those cases we actually don't care about this value - it's
	// not returned in the user-facing models (we do use it in other contexts, and the value is valid there).
	// So we'll just fallback to letting it be time.Zero.
	if timeStr == "0001-01-01T00:00:00" {
		return time.Time{}, nil
	}

	parsedTime, err := time.Parse(time.RFC3339, timeStr)

	if err != nil {
		return time.Time{}, err
	}

	return parsedTime, nil
}
