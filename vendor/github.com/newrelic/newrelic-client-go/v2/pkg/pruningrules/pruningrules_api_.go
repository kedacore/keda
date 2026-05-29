package pruningrules

import (
	"context"
	"fmt"
)

// GetPruningRuleByID fetches a single pruning rule by ID from the account's rule list.
func (a *Pruningrules) GetPruningRuleByID(
	accountID int,
	ruleID string,
) (*NRQLDropRulesDropRule, error) {
	result, err := a.GetListWithContext(context.Background(), accountID)
	if err != nil {
		return nil, err
	}
	for _, rule := range result.Rules {
		if rule.ID == ruleID {
			return &rule, nil
		}
	}
	return nil, fmt.Errorf("pruning rule with ID %s not found", ruleID)
}
