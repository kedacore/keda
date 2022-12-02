package entities

import (
	"encoding/json"
	"errors"

	log "github.com/sirupsen/logrus"
)

// Need Outlines to also implement Entity
func (x AlertableEntityOutline) ImplementsEntity()                       {}
func (x ApmApplicationEntityOutline) ImplementsEntity()                  {}
func (x ApmBrowserApplicationEntityOutline) ImplementsEntity()           {}
func (x ApmDatabaseInstanceEntityOutline) ImplementsEntity()             {}
func (x ApmExternalServiceEntityOutline) ImplementsEntity()              {}
func (x BrowserApplicationEntityOutline) ImplementsEntity()              {}
func (x DashboardEntityOutline) ImplementsEntity()                       {}
func (x EntityOutline) ImplementsEntity()                                {}
func (x GenericEntityOutline) ImplementsEntity()                         {}
func (x GenericInfrastructureEntityOutline) ImplementsEntity()           {}
func (x InfrastructureAwsLambdaFunctionEntityOutline) ImplementsEntity() {}
func (x InfrastructureHostEntityOutline) ImplementsEntity()              {}
func (x InfrastructureIntegrationEntityOutline) ImplementsEntity()       {}
func (x MobileApplicationEntityOutline) ImplementsEntity()               {}
func (x SecureCredentialEntityOutline) ImplementsEntity()                {}
func (x SyntheticMonitorEntityOutline) ImplementsEntity()                {}
func (x ThirdPartyServiceEntityOutline) ImplementsEntity()               {}
func (x UnavailableEntityOutline) ImplementsEntity()                     {}
func (x WorkloadEntityOutline) ImplementsEntity()                        {}

// UnmarshalJSON is used to unmarshal Actor into the correct
// interfaces per field.
func (a *Actor) UnmarshalJSON(b []byte) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(b, &objMap)
	if err != nil {
		return err
	}

	for k, v := range objMap {
		if v == nil {
			continue
		}

		switch k {
		case "entity":
			var e *EntityInterface
			e, err = UnmarshalEntityInterface([]byte(*v))
			if err != nil {
				return err
			}

			a.Entity = *e
		case "entities":
			var rawEntities []*json.RawMessage
			err = json.Unmarshal(*v, &rawEntities)
			if err != nil {
				return err
			}

			for _, m := range rawEntities {
				var e *EntityInterface
				e, err = UnmarshalEntityInterface(*m)
				if err != nil {
					return err
				}

				if e != nil {
					a.Entities = append(a.Entities, *e)
				}
			}
		case "entitySearch":
			var es EntitySearch
			err = json.Unmarshal(*v, &es)
			if err != nil {
				return err
			}

			a.EntitySearch = es
		default:
			log.Errorf("Unknown key '%s' value: %s", k, string(*v))
		}
	}

	return nil
}

// MarshalJSON returns the JSON encoded version of DashboardWidgetRawConfiguration
// (which is already JSON)
func (d DashboardWidgetRawConfiguration) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("null"), nil
	}

	return d, nil
}

// UnmarshalJSON sets *d to a copy of the data, as DashboardWidgetRawConfiguration is
// the raw JSON intentionally.
func (d *DashboardWidgetRawConfiguration) UnmarshalJSON(data []byte) error {
	if d == nil {
		return errors.New("entities.DashboardWidgetRawConfiguration: UnmarshalJSON on nil pointer")
	}

	*d = append((*d)[0:0], data...)
	return nil
}
