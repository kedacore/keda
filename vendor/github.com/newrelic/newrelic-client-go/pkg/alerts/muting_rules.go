package alerts

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MutingRule represents the alert suppression mechanism in the Alerts API.
type MutingRule struct {
	ID            int                      `json:"id,string,omitempty"`
	AccountID     int                      `json:"accountId,omitempty"`
	Condition     MutingRuleConditionGroup `json:"condition,omitempty"`
	CreatedAt     string                   `json:"createdAt,omitempty"`
	CreatedByUser ByUser                   `json:"createdByUser,omitempty"`
	Description   string                   `json:"description,omitempty"`
	Enabled       bool                     `json:"enabled"`
	Name          string                   `json:"name,omitempty"`
	UpdatedAt     string                   `json:"updatedAt,omitempty"`
	UpdatedByUser ByUser                   `json:"updatedByUser,omitempty"`
	Schedule      *MutingRuleSchedule      `json:"schedule,omitempty"`
}

// ByUser is a collection of the user information that created or updated the muting rule.
type ByUser struct {
	Email    string `json:"email"`
	Gravatar string `json:"gravatar"`
	ID       int    `json:"id"`
	Name     string `json:"name"`
}

// MutingRuleConditionGroup is a collection of conditions for muting.
type MutingRuleConditionGroup struct {
	Conditions []MutingRuleCondition `json:"conditions"`
	Operator   string                `json:"operator"`
}

// MutingRuleCondition is a single muting rule condition.
type MutingRuleCondition struct {
	Attribute string   `json:"attribute"`
	Operator  string   `json:"operator"`
	Values    []string `json:"values"`
}

// MutingRuleScheduleRepeat represents how frequently a MutingRule's schedule repeats.
type MutingRuleScheduleRepeat string

// MutingRuleScheduleRepeatTypes are intervals for MutingRulesScheduleRepeat.
var MutingRuleScheduleRepeatTypes = struct {
	// DAILY - Schedule repeats once per calendar day.
	DAILY MutingRuleScheduleRepeat
	// WEEKLY - Schedule repeats once per specified day per calendar week.
	WEEKLY MutingRuleScheduleRepeat
	// MONTHLY - Schedule repeats once per calendar month.
	MONTHLY MutingRuleScheduleRepeat
}{
	DAILY:   "DAILY",
	WEEKLY:  "WEEKLY",
	MONTHLY: "MONTHLY",
}

// DayOfWeek is used to configure a WEEKLY scheduled MutingRule.
type DayOfWeek string

// DayOfWeekTypes are days of the week for DayOfWeek.
var DayOfWeekTypes = struct {
	MONDAY    DayOfWeek
	TUESDAY   DayOfWeek
	WEDNESDAY DayOfWeek
	THURSDAY  DayOfWeek
	FRIDAY    DayOfWeek
	SATURDAY  DayOfWeek
	SUNDAY    DayOfWeek
}{
	MONDAY:    "MONDAY",
	TUESDAY:   "TUESDAY",
	WEDNESDAY: "WEDNESDAY",
	THURSDAY:  "THURSDAY",
	FRIDAY:    "FRIDAY",
	SATURDAY:  "SATURDAY",
	SUNDAY:    "SUNDAY",
}

// NaiveDateTime wraps `time.Time` to remove the time zone offset when JSON marshaling.
// NaiveDateTime is used for MutingRuleScheduleCreateInput and MutingRuleScheduleUpdateInput fields StartTime, EndTime, and EndRepeat.
type NaiveDateTime struct {
	time.Time
}

// MarshalJSON strips the UTC time zone offset from the NaiveDateTime when JSON marshaling.
// If a non-UTC time zone offset is specified on the NaiveDateTime, an error will be thrown.
func (t NaiveDateTime) MarshalJSON() ([]byte, error) {
	if _, offset := t.Zone(); offset != 0 {
		return nil, fmt.Errorf("time offset %d not allowed. You can call .UTC() on the time provided to reset the offset", offset)
	}

	return json.Marshal(t.Format("2006-01-02T15:04:05"))
}

// MutingRuleSchedule is the time window when the MutingRule should actively mute violations
type MutingRuleSchedule struct {
	StartTime        *time.Time                `json:"startTime,omitempty"`
	EndTime          *time.Time                `json:"endTime,omitempty"`
	TimeZone         string                    `json:"timeZone"`
	Repeat           *MutingRuleScheduleRepeat `json:"repeat,omitempty"`
	EndRepeat        *time.Time                `json:"endRepeat,omitempty"`
	RepeatCount      *int                      `json:"repeatCount,omitempty"`
	WeeklyRepeatDays *[]DayOfWeek              `json:"weeklyRepeatDays,omitempty"`
}

// MutingRuleScheduleCreateInput is the time window when the MutingRule should actively mute violations for Create
type MutingRuleScheduleCreateInput struct {
	StartTime        *NaiveDateTime            `json:"startTime,omitempty"`
	EndTime          *NaiveDateTime            `json:"endTime,omitempty"`
	TimeZone         string                    `json:"timeZone"`
	Repeat           *MutingRuleScheduleRepeat `json:"repeat,omitempty"`
	EndRepeat        *NaiveDateTime            `json:"endRepeat,omitempty"`
	RepeatCount      *int                      `json:"repeatCount,omitempty"`
	WeeklyRepeatDays *[]DayOfWeek              `json:"weeklyRepeatDays,omitempty"`
}

// MutingRuleScheduleUpdateInput is the time window when the MutingRule should actively mute violations for Update
type MutingRuleScheduleUpdateInput struct {
	StartTime        *NaiveDateTime            `json:"startTime"`
	EndTime          *NaiveDateTime            `json:"endTime"`
	TimeZone         *string                   `json:"timeZone"`
	Repeat           *MutingRuleScheduleRepeat `json:"repeat"`
	EndRepeat        *NaiveDateTime            `json:"endRepeat"`
	RepeatCount      *int                      `json:"repeatCount"`
	WeeklyRepeatDays *[]DayOfWeek              `json:"weeklyRepeatDays"`
}

// MutingRuleCreateInput is the input for creating muting rules.
type MutingRuleCreateInput struct {
	Condition   MutingRuleConditionGroup       `json:"condition"`
	Description string                         `json:"description"`
	Enabled     bool                           `json:"enabled"`
	Name        string                         `json:"name"`
	Schedule    *MutingRuleScheduleCreateInput `json:"schedule,omitempty"`
}

// MutingRuleUpdateInput is the input for updating a rule.
type MutingRuleUpdateInput struct {
	// Condition is is available from the API, but the json needs to be handled
	// properly.

	Condition   *MutingRuleConditionGroup      `json:"condition,omitempty"`
	Description string                         `json:"description,omitempty"`
	Enabled     bool                           `json:"enabled"`
	Name        string                         `json:"name,omitempty"`
	Schedule    *MutingRuleScheduleUpdateInput `json:"schedule"`
}

// ListMutingRules queries for all muting rules in a given account.
func (a *Alerts) ListMutingRules(accountID int) ([]MutingRule, error) {
	return a.ListMutingRulesWithContext(context.Background(), accountID)
}

// ListMutingRulesWithContext queries for all muting rules in a given account.
func (a *Alerts) ListMutingRulesWithContext(ctx context.Context, accountID int) ([]MutingRule, error) {
	vars := map[string]interface{}{
		"accountID": accountID,
	}

	resp := alertMutingRuleListResponse{}

	if err := a.client.NerdGraphQueryWithContext(ctx, alertsMutingRulesQuery, vars, &resp); err != nil {
		return nil, err
	}

	return resp.Actor.Account.Alerts.MutingRules, nil
}

// GetMutingRule queries for a single muting rule matching the given ID.
func (a *Alerts) GetMutingRule(accountID, ruleID int) (*MutingRule, error) {
	return a.GetMutingRuleWithContext(context.Background(), accountID, ruleID)
}

// GetMutingRuleWithContext queries for a single muting rule matching the given ID.
func (a *Alerts) GetMutingRuleWithContext(ctx context.Context, accountID, ruleID int) (*MutingRule, error) {
	vars := map[string]interface{}{
		"accountID": accountID,
		"ruleID":    ruleID,
	}

	resp := alertMutingRulesGetResponse{}

	if err := a.NerdGraphQueryWithContext(ctx, alertsMutingRulesGet, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.Actor.Account.Alerts.MutingRule, nil
}

// CreateMutingRule is the mutation to create a muting rule for the given account and input.
func (a *Alerts) CreateMutingRule(accountID int, rule MutingRuleCreateInput) (*MutingRule, error) {
	return a.CreateMutingRuleWithContext(context.Background(), accountID, rule)
}

// CreateMutingRuleWithContext is the mutation to create a muting rule for the given account and input.
func (a *Alerts) CreateMutingRuleWithContext(ctx context.Context, accountID int, rule MutingRuleCreateInput) (*MutingRule, error) {
	vars := map[string]interface{}{
		"accountID": accountID,
		"rule":      rule,
	}

	resp := alertMutingRuleCreateResponse{}

	if err := a.NerdGraphQueryWithContext(ctx, alertsMutingRulesCreate, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AlertsMutingRuleCreate, nil
}

// UpdateMutingRule is the mutation to update an existing muting rule.
func (a *Alerts) UpdateMutingRule(accountID int, ruleID int, rule MutingRuleUpdateInput) (*MutingRule, error) {
	return a.UpdateMutingRuleWithContext(context.Background(), accountID, ruleID, rule)
}

// UpdateMutingRuleWithContext is the mutation to update an existing muting rule.
func (a *Alerts) UpdateMutingRuleWithContext(ctx context.Context, accountID int, ruleID int, rule MutingRuleUpdateInput) (*MutingRule, error) {
	vars := map[string]interface{}{
		"accountID": accountID,
		"ruleID":    ruleID,
		"rule":      rule,
	}

	resp := alertMutingRuleUpdateResponse{}

	if err := a.NerdGraphQueryWithContext(ctx, alertsMutingRulesUpdate, vars, &resp); err != nil {
		return nil, err
	}

	return &resp.AlertsMutingRuleUpdate, nil
}

// DeleteMutingRule is the mutation to delete an existing muting rule.
func (a *Alerts) DeleteMutingRule(accountID int, ruleID int) error {
	return a.DeleteMutingRuleWithContext(context.Background(), accountID, ruleID)
}

// DeleteMutingRuleWithContext is the mutation to delete an existing muting rule.
func (a *Alerts) DeleteMutingRuleWithContext(ctx context.Context, accountID int, ruleID int) error {
	vars := map[string]interface{}{
		"accountID": accountID,
		"ruleID":    ruleID,
	}

	resp := alertMutingRuleDeleteResponse{}

	return a.NerdGraphQueryWithContext(ctx, alertsMutingRuleDelete, vars, &resp)
}

type alertMutingRuleCreateResponse struct {
	AlertsMutingRuleCreate MutingRule `json:"alertsMutingRuleCreate"`
}

type alertMutingRuleUpdateResponse struct {
	AlertsMutingRuleUpdate MutingRule `json:"alertsMutingRuleUpdate"`
}

type alertMutingRuleDeleteResponse struct {
	AlertsMutingRuleDelete struct {
		ID string `json:"id"`
	} `json:"alertsMutingRuleDelete"`
}

type alertMutingRuleListResponse struct {
	Actor struct {
		Account struct {
			Alerts struct {
				MutingRules []MutingRule `json:"mutingRules"`
			} `json:"alerts"`
		} `json:"account"`
	} `json:"actor"`
}

type alertMutingRulesGetResponse struct {
	Actor struct {
		Account struct {
			Alerts struct {
				MutingRule MutingRule `json:"mutingRule"`
			} `json:"alerts"`
		} `json:"account"`
	} `json:"actor"`
}

const (
	alertsMutingRulesQuery = `query($accountID: Int!) {
		actor {
			account(id: $accountID) {
				alerts {
					mutingRules {
						id
						name
						description
						enabled
						condition {
							operator
							conditions {
								attribute
								operator
								values
							}
						}
					}
				}
			}
		}
	}`

	alertsMutingRulesGet = `query($accountID: Int!, $ruleID: ID!) {
		actor {
			account(id: $accountID) {
				alerts {
					mutingRule(id: $ruleID) {` +
		alertsMutingRuleFields +
		`}}}}}`

	alertsMutingRuleFields = `
		accountId
		condition {
			conditions {
				attribute
				operator
				values
			}
			operator
		}
		id
		name
		enabled
		description
		createdAt
		createdByUser {
			email
			gravatar
			id
			name
		}
		updatedAt
		updatedByUser {
			email
			gravatar
			id
			name
		}
		schedule {
			startTime
			endTime
			timeZone
			repeat
			endRepeat
			repeatCount
			weeklyRepeatDays
		}
	`

	alertsMutingRulesCreate = `mutation CreateRule($accountID: Int!, $rule: AlertsMutingRuleInput!) {
		alertsMutingRuleCreate(accountId: $accountID, rule: $rule) {` +
		alertsMutingRuleFields +

		`}
	}`

	alertsMutingRulesUpdate = `mutation UpdateRule($accountID: Int!, $ruleID: ID!, $rule: AlertsMutingRuleUpdateInput!) {
		alertsMutingRuleUpdate(accountId: $accountID, id: $ruleID, rule: $rule) {` +
		alertsMutingRuleFields +
		`}
	}`

	alertsMutingRuleDelete = `mutation DeleteRule($accountID: Int!, $ruleID: ID!) {
		alertsMutingRuleDelete(accountId: $accountID, id: $ruleID) {
			id
		}
	}`
)
