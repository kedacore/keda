// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package admin

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/runtime"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/messaging/azservicebus/internal/atom"
)

// Rule specifies a message filter and action for a subscription.
type Rule struct {
	// Filter is the filter that will be used for Rule.
	// Valid types: *SQLFilter, *CorrelationFilter, *FalseFilter, *TrueFilter
	Filter RuleFilter

	// Action is the action that will be used for Rule.
	// Valid types: *SQLAction
	Action RuleAction
}

// RuleFilter is a filter for a subscription rule.
// Implemented by: *SQLFilter, *CorrelationFilter, *FalseFilter, *TrueFilter
type RuleFilter interface {
	ruleFilter()
}

// RuleAction is an action for a subscription rule.
// Implemented by: *SQLAction
type RuleAction interface {
	ruleAction()
}

// SQLAction is an action that updates a message according to its
// expression.
type SQLAction struct {
	// Expression is a SQL Expression
	Expression string

	// Parameters is a map of string to values of type string, number, or boolean.
	Parameters map[string]any
}

func (a *SQLAction) ruleAction() {}

// UnknownRuleAction is an action type not yet handled by this SDK.
// If you get this type back you should update to a newer version of the SDK
// which properly represents this type.
type UnknownRuleAction struct {
	// Type is the Service Bus type for this action.
	Type string

	// RawXML is the raw XML for this action that could not be parsed.
	RawXML []byte
}

func (a *UnknownRuleAction) ruleAction() {}

// SQLFilter is a filter that evaluates to true for any message that matches
// its expression.
type SQLFilter struct {
	// Expression is a SQL Expression
	Expression string

	// Parameters is a map of string to values of type string, number, or boolean.
	Parameters map[string]any
}

func (f *SQLFilter) ruleFilter() {}

// TrueFilter is a filter that always evaluates to true for any message.
type TrueFilter struct{}

func (f *TrueFilter) ruleFilter() {}

// FalseFilter is a filter that always evaluates to false for any message.
type FalseFilter struct{}

func (f *FalseFilter) ruleFilter() {}

// CorrelationFilter represents a set of conditions that are matched against user
// and system properties of messages for a subscription.
type CorrelationFilter struct {
	// ApplicationProperties will be matched against the application properties for the message.
	ApplicationProperties map[string]any

	// ContentType will be matched against the ContentType property for the message.
	ContentType *string

	// CorrelationID will be matched against the CorrelationID property for the message.
	CorrelationID *string

	// MessageID will be matched against the MessageID property for the message.
	MessageID *string

	// ReplyTo will be matched against the ReplyTo property for the message.
	ReplyTo *string

	// ReplyToSessionID will be matched against the ReplyToSessionID property for the message.
	ReplyToSessionID *string

	// SessionID will be matched against the SessionID property for the message.
	SessionID *string

	// Subject will be matched against the Subject property for the message.
	Subject *string

	// To will be matched against the To property for the message.
	To *string
}

func (f *CorrelationFilter) ruleFilter() {}

// UnknownRuleFilter is a filter type not yet handled by this SDK.
// If you get this type back you should update to a newer version of the SDK
// which properly represents this type.
type UnknownRuleFilter struct {
	// Type is the Service Bus type for this filter.
	Type string

	// RawXML is the raw XML for this rule that could not be parsed.
	RawXML []byte
}

func (f *UnknownRuleFilter) ruleFilter() {}

// RuleProperties are the properties for a rule.
type RuleProperties struct {
	// Name is the name of this rule.
	Name string

	// Filter is the filter that will be used for Rule.
	// Valid types: *SQLFilter, *CorrelationFilter, *FalseFilter, *TrueFilter
	Filter RuleFilter

	// Action is the action that will be used for Rule.
	// Valid types: *SQLAction
	Action RuleAction
}

// CreateRuleResponse contains the response fields for Client.CreateRule
type CreateRuleResponse struct {
	RuleProperties
}

// CreateRuleOptions contains the optional parameters for Client.CreateRule
type CreateRuleOptions struct {
	// Name is the name of the rule or nil, which will default to $Default
	Name *string

	// Filter is the filter that will be used for Rule.
	// Valid types: *SQLFilter, *CorrelationFilter, *FalseFilter, *TrueFilter
	Filter RuleFilter

	// Action is the action that will be used for Rule.
	// Valid types: *SQLAction
	Action RuleAction
}

// CreateRule creates a rule that can filter and update message for a subscription.
func (ac *Client) CreateRule(ctx context.Context, topicName string, subscriptionName string, options *CreateRuleOptions) (CreateRuleResponse, error) {
	ruleName := ""

	if options != nil && options.Name != nil {
		ruleName = *options.Name
	}

	resp, _, err := ac.createOrUpdateRule(ctx, topicName, subscriptionName, RuleProperties{
		Name:   ruleName,
		Filter: options.Filter,
		Action: options.Action,
	}, true)

	if err != nil {
		return CreateRuleResponse{}, err
	}

	return CreateRuleResponse{RuleProperties: *resp}, nil
}

// GetRuleResponse contains the response fields for Client.GetRule
type GetRuleResponse struct {
	// RuleProperties for the rule.
	RuleProperties
}

// GetRuleOptions contains the optional parameters for Client.GetRule
type GetRuleOptions struct {
	// For future expansion
}

// GetRule gets a rule for a subscription.
func (ac *Client) GetRule(ctx context.Context, topicName string, subscriptionName string, ruleName string, options *GetRuleOptions) (*GetRuleResponse, error) {
	var ruleEnv *atom.RuleEnvelope

	_, err := ac.em.Get(ctx, fmt.Sprintf("/%s/Subscriptions/%s/Rules/%s", topicName, subscriptionName, ruleName), &ruleEnv)

	if err != nil {
		return mapATOMError[GetRuleResponse](err)
	}

	props, err := ac.newRuleProperties(ruleEnv)

	if err != nil {
		return nil, err
	}

	return &GetRuleResponse{
		RuleProperties: *props,
	}, nil
}

// ListRulesResponse contains the response fields for the pager returned from Client.ListRules.
type ListRulesResponse struct {
	// Rules are all the rules for the page.
	Rules []RuleProperties
}

// ListRulesOptions contains the optional parameters for Client.ListRules
type ListRulesOptions struct {
	// MaxPageSize is the maximum size of each page of results.
	MaxPageSize int32
}

// NewListRulesPager creates a pager that can list rules for a subscription.
func (ac *Client) NewListRulesPager(topicName string, subscriptionName string, options *ListRulesOptions) *runtime.Pager[ListRulesResponse] {
	var pageSize int32

	if options != nil {
		pageSize = options.MaxPageSize
	}

	ep := &entityPager[atom.RuleFeed, atom.RuleEnvelope, RuleProperties]{
		convertFn:    ac.newRuleProperties,
		baseFragment: fmt.Sprintf("/%s/Subscriptions/%s/Rules/", topicName, subscriptionName),
		maxPageSize:  pageSize,
		em:           ac.em,
	}

	return runtime.NewPager(runtime.PagingHandler[ListRulesResponse]{
		More: func(ltr ListRulesResponse) bool {
			return ep.More()
		},
		Fetcher: func(ctx context.Context, t *ListRulesResponse) (ListRulesResponse, error) {
			items, err := ep.Fetcher(ctx)

			if err != nil {
				return ListRulesResponse{}, err
			}

			return ListRulesResponse{
				Rules: items,
			}, nil
		},
	})
}

// UpdateRuleResponse contains the response fields for Client.UpdateRule
type UpdateRuleResponse struct {
	// RuleProperties for the updated rule.
	RuleProperties
}

// UpdateRuleOptions can be used to configure the UpdateRule method.
type UpdateRuleOptions struct {
	// For future expansion
}

// UpdateRule updates a rule for a subscription.
func (ac *Client) UpdateRule(ctx context.Context, topicName string, subscriptionName string, properties RuleProperties) (UpdateRuleResponse, error) {
	resp, _, err := ac.createOrUpdateRule(ctx, topicName, subscriptionName, properties, false)

	if err != nil {
		return UpdateRuleResponse{}, err
	}

	return UpdateRuleResponse{RuleProperties: *resp}, nil
}

// DeleteRuleResponse contains the response fields for Client.DeleteRule
type DeleteRuleResponse struct {
	// For future expansion
}

// DeleteRuleOptions can be used to configure the Client.DeleteRule method.
type DeleteRuleOptions struct {
	// For future expansion
}

// DeleteRule deletes a rule for a subscription.
func (ac *Client) DeleteRule(ctx context.Context, topicName string, subscriptionName string, ruleName string, options *DeleteRuleOptions) (DeleteRuleResponse, error) {
	_, err := ac.em.Delete(ctx, fmt.Sprintf("/%s/Subscriptions/%s/Rules/%s", topicName, subscriptionName, ruleName))

	return DeleteRuleResponse{}, err
}

func (ac *Client) createOrUpdateRule(ctx context.Context, topicName string, subscriptionName string, putProps RuleProperties, creating bool) (*RuleProperties, *http.Response, error) {
	ruleDesc := atom.RuleDescription{}

	theirFilter := putProps.Filter

	if theirFilter != nil {
		switch actualFilter := theirFilter.(type) {
		case *FalseFilter:
			ruleDesc.Filter = &atom.FilterDescription{
				Type:          "FalseFilter",
				SQLExpression: to.Ptr("1=0"),
			}
		case *TrueFilter:
			ruleDesc.Filter = &atom.FilterDescription{
				Type:          "TrueFilter",
				SQLExpression: to.Ptr("1=1"),
			}
		case *SQLFilter:
			params, err := publicSQLParametersToInternal(actualFilter.Parameters)

			if err != nil {
				return nil, nil, err
			}

			ruleDesc.Filter = &atom.FilterDescription{
				Type:          "SqlFilter",
				SQLExpression: &actualFilter.Expression,
				Parameters:    params,
			}
		case *CorrelationFilter:
			appProps, err := publicSQLParametersToInternal(actualFilter.ApplicationProperties)

			if err != nil {
				return nil, nil, err
			}

			ruleDesc.Filter = &atom.FilterDescription{
				Type: "CorrelationFilter",
				CorrelationFilter: atom.CorrelationFilter{
					ContentType:      actualFilter.ContentType,
					CorrelationID:    actualFilter.CorrelationID,
					MessageID:        actualFilter.MessageID,
					ReplyTo:          actualFilter.ReplyTo,
					ReplyToSessionID: actualFilter.ReplyToSessionID,
					SessionID:        actualFilter.SessionID,
					Label:            actualFilter.Subject,
					To:               actualFilter.To,
					Properties:       appProps,
				},
			}
		case *UnknownRuleFilter:
			fd, err := convertUnknownRuleFilterToFilterDescription(actualFilter)

			if err != nil {
				return nil, nil, err
			}

			ruleDesc.Filter = fd
		default:
			return nil, nil, fmt.Errorf("invalid type ('%T') for Rule.Filter", theirFilter)
		}
	} else {
		ruleDesc.Filter = &atom.FilterDescription{
			Type:          "TrueFilter",
			SQLExpression: to.Ptr("1=1"),
		}
	}

	theirAction := putProps.Action

	if theirAction != nil {
		switch actualAction := theirAction.(type) {
		case *SQLAction:
			params, err := publicSQLParametersToInternal(actualAction.Parameters)

			if err != nil {
				return nil, nil, err
			}

			ruleDesc.Action = &atom.ActionDescription{
				Type:          "SqlRuleAction",
				SQLExpression: actualAction.Expression,
				Parameters:    params,
			}
		case *UnknownRuleAction:
			ad, err := convertUnknownRuleActionToActionDescription(actualAction)

			if err != nil {
				return nil, nil, err
			}

			ruleDesc.Action = ad
		default:
			return nil, nil, fmt.Errorf("invalid type ('%T') for Rule.Action", theirAction)
		}
	}

	ruleDesc.Name = "$Default"

	if putProps.Name != "" {
		ruleDesc.Name = putProps.Name
	}

	if !creating {
		ctx = runtime.WithHTTPHeader(ctx, http.Header{
			"If-Match": []string{"*"},
		})
	}

	putEnv := atom.WrapWithRuleEnvelope(&ruleDesc)

	var respEnv *atom.RuleEnvelope

	httpResp, err := ac.em.Put(ctx, fmt.Sprintf("/%s/Subscriptions/%s/Rules/%s", topicName, subscriptionName, putProps.Name), putEnv, &respEnv, nil)

	if err != nil {
		return nil, nil, err
	}

	respProps, err := ac.newRuleProperties(respEnv)

	return respProps, httpResp, err
}

func (ac *Client) newRuleProperties(env *atom.RuleEnvelope) (*RuleProperties, error) {
	desc := env.Content.RuleDescription

	props := RuleProperties{
		Name: env.Title,
	}

	switch desc.Filter.Type {
	case "TrueFilter":
		props.Filter = &TrueFilter{}
	case "FalseFilter":
		props.Filter = &FalseFilter{}
	case "CorrelationFilter":
		cf := desc.Filter.CorrelationFilter

		appProps, err := internalSQLParametersToPublic(cf.Properties)

		if err != nil {
			return nil, err
		}

		props.Filter = &CorrelationFilter{
			ContentType:           cf.ContentType,
			CorrelationID:         cf.CorrelationID,
			MessageID:             cf.MessageID,
			ReplyTo:               cf.ReplyTo,
			ReplyToSessionID:      cf.ReplyToSessionID,
			SessionID:             cf.SessionID,
			Subject:               cf.Label,
			To:                    cf.To,
			ApplicationProperties: appProps,
		}
	case "SqlFilter":
		params, err := internalSQLParametersToPublic(desc.Filter.Parameters)

		if err != nil {
			return nil, err
		}

		props.Filter = &SQLFilter{
			Expression: *desc.Filter.SQLExpression,
			Parameters: params,
		}
	default:
		urf, err := newUnknownRuleFilterFromFilterDescription(desc.Filter)

		if err != nil {
			return nil, err
		}

		props.Filter = urf
	}

	const emptyRuleAction = "EmptyRuleAction"

	switch desc.Action.Type {
	case emptyRuleAction:
	case "SqlRuleAction":
		params, err := internalSQLParametersToPublic(desc.Action.Parameters)

		if err != nil {
			return nil, err
		}

		props.Action = &SQLAction{
			Expression: desc.Action.SQLExpression,
			Parameters: params,
		}
	default:
		ura, err := newUnknownRuleActionFromActionDescription(desc.Action)

		if err != nil {
			return nil, err
		}

		props.Action = ura
	}

	return &props, nil
}

func publicSQLParametersToInternal(publicParams map[string]any) (*atom.KeyValueList, error) {
	if len(publicParams) == 0 {
		return nil, nil
	}

	var params []atom.KeyValueOfstringanyType

	for k, v := range publicParams {
		switch asType := v.(type) {
		case string:
			params = append(params, atom.KeyValueOfstringanyType{
				Key: k,
				Value: atom.Value{
					Type:  "l28:string",
					L28NS: "http://www.w3.org/2001/XMLSchema",
					Text:  asType,
				},
			})
		case bool:
			params = append(params, atom.KeyValueOfstringanyType{
				Key: k,
				Value: atom.Value{
					Type:  "l28:boolean",
					L28NS: "http://www.w3.org/2001/XMLSchema",
					Text:  fmt.Sprintf("%t", v),
				},
			})
		case int, int64, int32:
			params = append(params, atom.KeyValueOfstringanyType{
				Key: k,
				Value: atom.Value{
					Type:  "l28:int",
					L28NS: "http://www.w3.org/2001/XMLSchema",
					Text:  fmt.Sprintf("%d", v),
				},
			})
		case float32, float64:
			params = append(params, atom.KeyValueOfstringanyType{
				Key: k,
				Value: atom.Value{
					Type:  "l28:double",
					L28NS: "http://www.w3.org/2001/XMLSchema",
					Text:  fmt.Sprintf("%f", v),
				},
			})
		case time.Time:
			params = append(params, atom.KeyValueOfstringanyType{
				Key: k,
				Value: atom.Value{
					Type:  "l28:dateTime",
					L28NS: "http://www.w3.org/2001/XMLSchema",
					Text:  asType.UTC().Format(time.RFC3339Nano),
				},
			})
		default:
			// TODO: 'duration'
			return nil, fmt.Errorf("type %T of parameter %s is not a handled type for SQL parameters", v, k)
		}
	}

	return &atom.KeyValueList{KeyValues: params}, nil
}

func internalSQLParametersToPublic(kvlist *atom.KeyValueList) (map[string]any, error) {
	if kvlist == nil {
		return nil, nil
	}

	params := map[string]any{}

	for _, p := range kvlist.KeyValues {
		// we only care about the actual type here since we can assume the
		// service is able to properly format/namespace its own XML
		valueType := p.Value.Type
		typeParts := strings.Split(p.Value.Type, ":")

		if len(typeParts) == 2 {
			valueType = typeParts[1]
		}

		switch valueType {
		case "string":
			params[p.Key] = p.Value.Text
		case "boolean":
			val, err := strconv.ParseBool(p.Value.Text)

			if err != nil {
				return nil, err
			}

			params[p.Key] = val
		case "int":
			val, err := strconv.ParseInt(p.Value.Text, 10, 64)

			if err != nil {
				return nil, err
			}

			params[p.Key] = val
		case "double":
			val, err := strconv.ParseFloat(p.Value.Text, 64)

			if err != nil {
				return nil, err
			}

			params[p.Key] = val
		case "dateTime":
			val, err := time.Parse(time.RFC3339Nano, p.Value.Text)

			if err != nil {
				return nil, err
			}

			params[p.Key] = val.UTC()
		default:
			// TODO: timespan
			return nil, fmt.Errorf("type %s of parameter %s is not a handled type for SQL parameters", valueType, p.Key)
		}
	}

	if len(params) == 0 {
		return nil, nil
	}

	return params, nil
}

func newUnknownRuleFilterFromFilterDescription(fd *atom.FilterDescription) (*UnknownRuleFilter, error) {
	attrs := fd.RawAttrs

	// 'type' gets parsed since it's one of the standard fields. Since we want to present
	// the full filter XML we'll re-add it.
	attrs = append(attrs, xml.Attr{
		Name: xml.Name{
			Local: "i:type",
		}, Value: fd.Type,
	})

	userFacingXML := struct {
		XMLName xml.Name   `xml:"Filter"`
		Attrs   []xml.Attr `xml:",any,attr"`
		XML     []byte     `xml:",innerxml"`
	}{
		Attrs: attrs,
		XML:   fd.RawXML,
	}

	xmlBytes, err := xml.Marshal(userFacingXML)

	if err != nil {
		return nil, err
	}

	return &UnknownRuleFilter{
		Type:   fd.Type,
		RawXML: xmlBytes,
	}, nil
}

func convertUnknownRuleFilterToFilterDescription(urf *UnknownRuleFilter) (*atom.FilterDescription, error) {
	var fdXML struct {
		Type  string     `xml:"i type,attr"`
		Attrs []xml.Attr `xml:",any,attr"`
		XML   []byte     `xml:",innerxml"`
	}

	if err := xml.Unmarshal([]byte(urf.RawXML), &fdXML); err != nil {
		return nil, err
	}

	return &atom.FilterDescription{
		Type:     fdXML.Type,
		RawAttrs: fdXML.Attrs,
		RawXML:   fdXML.XML,
	}, nil
}

func newUnknownRuleActionFromActionDescription(ad *atom.ActionDescription) (*UnknownRuleAction, error) {
	attrs := ad.RawAttrs

	// 'type' gets parsed since it's one of the standard fields. Since we want to present
	// the full filter XML we'll re-add it.
	attrs = append(attrs, xml.Attr{
		Name: xml.Name{
			Local: "i:type",
		}, Value: ad.Type,
	})

	userFacingXML := struct {
		XMLName xml.Name   `xml:"Action"`
		Attrs   []xml.Attr `xml:",any,attr"`
		XML     []byte     `xml:",innerxml"`
	}{
		Attrs: attrs,
		XML:   ad.RawXML,
	}

	xmlBytes, err := xml.Marshal(userFacingXML)

	if err != nil {
		return nil, err
	}

	return &UnknownRuleAction{
		Type:   ad.Type,
		RawXML: xmlBytes,
	}, nil
}

// convertUnknownRuleActionToActionDescription creates an atom.ActionDescription.
// This XML was originally
func convertUnknownRuleActionToActionDescription(urf *UnknownRuleAction) (*atom.ActionDescription, error) {
	var adXML struct {
		Type  string     `xml:"i type,attr"`
		Attrs []xml.Attr `xml:",any,attr"`
		XML   []byte     `xml:",innerxml"`
	}

	if err := xml.Unmarshal([]byte(urf.RawXML), &adXML); err != nil {
		return nil, err
	}

	return &atom.ActionDescription{
		Type:     adXML.Type,
		RawXML:   adXML.XML,
		RawAttrs: adXML.Attrs,
	}, nil
}
