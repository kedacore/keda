package kafka

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/segmentio/kafka-go/protocol/createacls"
)

// CreateACLsRequest represents a request sent to a kafka broker to add
// new ACLs.
type CreateACLsRequest struct {
	// Address of the kafka broker to send the request to.
	Addr net.Addr

	// List of ACL to create.
	ACLs []ACLEntry
}

// CreateACLsResponse represents a response from a kafka broker to an ACL
// creation request.
type CreateACLsResponse struct {
	// The amount of time that the broker throttled the request.
	Throttle time.Duration

	// List of errors that occurred while attempting to create
	// the ACLs.
	//
	// The errors contain the kafka error code. Programs may use the standard
	// errors.Is function to test the error against kafka error codes.
	Errors []error
}

type ACLPermissionType int8

const (
	ACLPermissionTypeUnknown ACLPermissionType = 0
	ACLPermissionTypeAny     ACLPermissionType = 1
	ACLPermissionTypeDeny    ACLPermissionType = 2
	ACLPermissionTypeAllow   ACLPermissionType = 3
)

func (apt ACLPermissionType) String() string {
	mapping := map[ACLPermissionType]string{
		ACLPermissionTypeUnknown: "Unknown",
		ACLPermissionTypeAny:     "Any",
		ACLPermissionTypeDeny:    "Deny",
		ACLPermissionTypeAllow:   "Allow",
	}
	s, ok := mapping[apt]
	if !ok {
		s = mapping[ACLPermissionTypeUnknown]
	}
	return s
}

// MarshalText transforms an ACLPermissionType into its string representation.
func (apt ACLPermissionType) MarshalText() ([]byte, error) {
	return []byte(apt.String()), nil
}

// UnmarshalText takes a string representation of the resource type and converts it to an ACLPermissionType.
func (apt *ACLPermissionType) UnmarshalText(text []byte) error {
	normalized := strings.ToLower(string(text))
	mapping := map[string]ACLPermissionType{
		"unknown": ACLPermissionTypeUnknown,
		"any":     ACLPermissionTypeAny,
		"deny":    ACLPermissionTypeDeny,
		"allow":   ACLPermissionTypeAllow,
	}
	parsed, ok := mapping[normalized]
	if !ok {
		*apt = ACLPermissionTypeUnknown
		return fmt.Errorf("cannot parse %s as an ACLPermissionType", normalized)
	}
	*apt = parsed
	return nil
}

type ACLOperationType int8

const (
	ACLOperationTypeUnknown         ACLOperationType = 0
	ACLOperationTypeAny             ACLOperationType = 1
	ACLOperationTypeAll             ACLOperationType = 2
	ACLOperationTypeRead            ACLOperationType = 3
	ACLOperationTypeWrite           ACLOperationType = 4
	ACLOperationTypeCreate          ACLOperationType = 5
	ACLOperationTypeDelete          ACLOperationType = 6
	ACLOperationTypeAlter           ACLOperationType = 7
	ACLOperationTypeDescribe        ACLOperationType = 8
	ACLOperationTypeClusterAction   ACLOperationType = 9
	ACLOperationTypeDescribeConfigs ACLOperationType = 10
	ACLOperationTypeAlterConfigs    ACLOperationType = 11
	ACLOperationTypeIdempotentWrite ACLOperationType = 12
)

func (aot ACLOperationType) String() string {
	mapping := map[ACLOperationType]string{
		ACLOperationTypeUnknown:         "Unknown",
		ACLOperationTypeAny:             "Any",
		ACLOperationTypeAll:             "All",
		ACLOperationTypeRead:            "Read",
		ACLOperationTypeWrite:           "Write",
		ACLOperationTypeCreate:          "Create",
		ACLOperationTypeDelete:          "Delete",
		ACLOperationTypeAlter:           "Alter",
		ACLOperationTypeDescribe:        "Describe",
		ACLOperationTypeClusterAction:   "ClusterAction",
		ACLOperationTypeDescribeConfigs: "DescribeConfigs",
		ACLOperationTypeAlterConfigs:    "AlterConfigs",
		ACLOperationTypeIdempotentWrite: "IdempotentWrite",
	}
	s, ok := mapping[aot]
	if !ok {
		s = mapping[ACLOperationTypeUnknown]
	}
	return s
}

// MarshalText transforms an ACLOperationType into its string representation.
func (aot ACLOperationType) MarshalText() ([]byte, error) {
	return []byte(aot.String()), nil
}

// UnmarshalText takes a string representation of the resource type and converts it to an ACLPermissionType.
func (aot *ACLOperationType) UnmarshalText(text []byte) error {
	normalized := strings.ToLower(string(text))
	mapping := map[string]ACLOperationType{
		"unknown":         ACLOperationTypeUnknown,
		"any":             ACLOperationTypeAny,
		"all":             ACLOperationTypeAll,
		"read":            ACLOperationTypeRead,
		"write":           ACLOperationTypeWrite,
		"create":          ACLOperationTypeCreate,
		"delete":          ACLOperationTypeDelete,
		"alter":           ACLOperationTypeAlter,
		"describe":        ACLOperationTypeDescribe,
		"clusteraction":   ACLOperationTypeClusterAction,
		"describeconfigs": ACLOperationTypeDescribeConfigs,
		"alterconfigs":    ACLOperationTypeAlterConfigs,
		"idempotentwrite": ACLOperationTypeIdempotentWrite,
	}
	parsed, ok := mapping[normalized]
	if !ok {
		*aot = ACLOperationTypeUnknown
		return fmt.Errorf("cannot parse %s as an ACLOperationType", normalized)
	}
	*aot = parsed
	return nil

}

type ACLEntry struct {
	ResourceType        ResourceType
	ResourceName        string
	ResourcePatternType PatternType
	Principal           string
	Host                string
	Operation           ACLOperationType
	PermissionType      ACLPermissionType
}

// CreateACLs sends ACLs creation request to a kafka broker and returns the
// response.
func (c *Client) CreateACLs(ctx context.Context, req *CreateACLsRequest) (*CreateACLsResponse, error) {
	acls := make([]createacls.RequestACLs, 0, len(req.ACLs))

	for _, acl := range req.ACLs {
		acls = append(acls, createacls.RequestACLs{
			ResourceType:        int8(acl.ResourceType),
			ResourceName:        acl.ResourceName,
			ResourcePatternType: int8(acl.ResourcePatternType),
			Principal:           acl.Principal,
			Host:                acl.Host,
			Operation:           int8(acl.Operation),
			PermissionType:      int8(acl.PermissionType),
		})
	}

	m, err := c.roundTrip(ctx, req.Addr, &createacls.Request{
		Creations: acls,
	})
	if err != nil {
		return nil, fmt.Errorf("kafka.(*Client).CreateACLs: %w", err)
	}

	res := m.(*createacls.Response)
	ret := &CreateACLsResponse{
		Throttle: makeDuration(res.ThrottleTimeMs),
		Errors:   make([]error, 0, len(res.Results)),
	}

	for _, t := range res.Results {
		ret.Errors = append(ret.Errors, makeError(t.ErrorCode, t.ErrorMessage))
	}

	return ret, nil
}
