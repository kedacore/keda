package nrqldroprules

import (
	"context"
	"fmt"
	"strconv"
)

// GetDropRuleByID helps to fetch a drop rule with the specified ID, from the list of drop rules retrieved for a given account
func (a *Nrqldroprules) GetDropRuleByID(
	accountID int,
	dropRuleID int,
) (*NRQLDropRulesDropRule, error) {
	dropRuleResult, err := a.GetListWithContext(context.Background(),
		accountID,
	)
	if err != nil {
		return nil, err
	}
	for _, dropRule := range dropRuleResult.Rules {
		if dropRule.ID == strconv.Itoa(dropRuleID) {
			return &dropRule, nil
		}
	}
	return nil, fmt.Errorf("drop rule with ID %d not found", dropRuleID)
}
