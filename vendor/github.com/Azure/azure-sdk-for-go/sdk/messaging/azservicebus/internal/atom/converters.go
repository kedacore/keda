// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package atom

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/auth"
)

func WrapWithQueueEnvelope(qd *QueueDescription, tokenProvider auth.TokenProvider) *QueueEnvelope {
	qd.ServiceBusSchema = to.Ptr(serviceBusSchema)

	qe := &QueueEnvelope{
		Entry: &Entry{
			AtomSchema: atomSchema,
		},
		Content: &QueueContent{
			Type:             applicationXML,
			QueueDescription: *qd,
		},
	}

	return qe
}

func WrapWithTopicEnvelope(td *TopicDescription) *TopicEnvelope {
	td.ServiceBusSchema = to.Ptr(serviceBusSchema)

	return &TopicEnvelope{
		Entry: &Entry{
			AtomSchema: atomSchema,
		},
		Content: &TopicContent{
			Type:             applicationXML,
			TopicDescription: *td,
		},
	}
}

func WrapWithSubscriptionEnvelope(sd *SubscriptionDescription) *SubscriptionEnvelope {
	sd.ServiceBusSchema = to.Ptr(serviceBusSchema)

	return &SubscriptionEnvelope{
		Entry: &Entry{
			AtomSchema: atomSchema,
		},
		Content: &SubscriptionContent{
			Type:                    applicationXML,
			SubscriptionDescription: *sd,
		},
	}
}

func WrapWithRuleEnvelope(rd *RuleDescription) *RuleEnvelope {
	rd.XMLNS = "http://schemas.microsoft.com/netservices/2010/10/servicebus/connect"
	rd.XMLNSI = "http://www.w3.org/2001/XMLSchema-instance"

	return &RuleEnvelope{
		Entry: &Entry{
			AtomSchema: atomSchema,
		},
		Content: &RuleContent{
			Type:            applicationXML,
			RuleDescription: *rd,
		},
	}
}
