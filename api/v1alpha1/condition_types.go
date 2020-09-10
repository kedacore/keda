package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConditionType specifies the available conditions for the resource
type ConditionType string

const (
	// ConditionReady specifies that the resource is ready.
	// For long-running resources.
	ConditionReady ConditionType = "Ready"
	// ConditionActive specifies that the resource has finished.
	// For resource which run to completion.
	ConditionActive ConditionType = "Active"
)

// Condition to store the condition state
type Condition struct {
	// Type of condition
	// +required
	Type ConditionType `json:"type" description:"type of status condition"`

	// Status of the condition, one of True, False, Unknown.
	// +required
	Status metav1.ConditionStatus `json:"status" description:"status of the condition, one of True, False, Unknown"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" description:"one-word CamelCase reason for the condition's last transition"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty" description:"human-readable message indicating details about last transition"`
}

// Conditions an array representation to store multiple Conditions
type Conditions []Condition

// AreInitialized performs check all Conditions are initialized
// return true if Conditions are initialized
// return false if Conditions are not initialized
func (c *Conditions) AreInitialized() bool {
	foundReady := false
	foundActive := false
	if *c != nil {
		for _, condition := range *c {
			if condition.Type == ConditionReady {
				foundReady = true
				break
			}
		}
		for _, condition := range *c {
			if condition.Type == ConditionActive {
				foundActive = true
				break
			}
		}
	}

	return foundReady && foundActive
}

// GetInitializedConditions returns Conditions initialized to the default -> Status: Unknown
func GetInitializedConditions() *Conditions {
	return &Conditions{{Type: ConditionReady, Status: metav1.ConditionUnknown}, {Type: ConditionActive, Status: metav1.ConditionUnknown}}
}

// IsTrue is true if the condition is True
func (c *Condition) IsTrue() bool {
	if c == nil {
		return false
	}
	return c.Status == metav1.ConditionTrue
}

// IsFalse is true if the condition is False
func (c *Condition) IsFalse() bool {
	if c == nil {
		return false
	}
	return c.Status == metav1.ConditionFalse
}

// IsUnknown is true if the condition is Unknown
func (c *Condition) IsUnknown() bool {
	if c == nil {
		return true
	}
	return c.Status == metav1.ConditionUnknown
}

// SetReadyCondition modifies Ready Condition according to input parameters
func (c *Conditions) SetReadyCondition(status metav1.ConditionStatus, reason string, message string) {
	if *c == nil {
		c = GetInitializedConditions()
	}
	c.setCondition(ConditionReady, status, reason, message)
}

// SetActiveCondition modifies Active Condition according to input parameters
func (c *Conditions) SetActiveCondition(status metav1.ConditionStatus, reason string, message string) {
	if *c == nil {
		c = GetInitializedConditions()
	}
	c.setCondition(ConditionActive, status, reason, message)
}

// GetActiveCondition returns Condition of type Active
func (c *Conditions) GetActiveCondition() Condition {
	if *c == nil {
		c = GetInitializedConditions()
	}
	return c.getCondition(ConditionActive)
}

func (c Conditions) getCondition(conditionType ConditionType) Condition {
	for i := range c {
		if c[i].Type == conditionType {
			return c[i]
		}
	}
	return Condition{}
}

func (c Conditions) setCondition(conditionType ConditionType, status metav1.ConditionStatus, reason string, message string) {
	for i := range c {
		if c[i].Type == conditionType {
			c[i].Status = status
			c[i].Reason = reason
			c[i].Message = message
			break
		}
	}
}
