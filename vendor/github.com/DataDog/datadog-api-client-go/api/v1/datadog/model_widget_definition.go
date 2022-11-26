// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// WidgetDefinition - [Definition of the widget](https://docs.datadoghq.com/dashboards/widgets/).
type WidgetDefinition struct {
	AlertGraphWidgetDefinition     *AlertGraphWidgetDefinition
	AlertValueWidgetDefinition     *AlertValueWidgetDefinition
	ChangeWidgetDefinition         *ChangeWidgetDefinition
	CheckStatusWidgetDefinition    *CheckStatusWidgetDefinition
	DistributionWidgetDefinition   *DistributionWidgetDefinition
	EventStreamWidgetDefinition    *EventStreamWidgetDefinition
	EventTimelineWidgetDefinition  *EventTimelineWidgetDefinition
	FreeTextWidgetDefinition       *FreeTextWidgetDefinition
	GeomapWidgetDefinition         *GeomapWidgetDefinition
	GroupWidgetDefinition          *GroupWidgetDefinition
	HeatMapWidgetDefinition        *HeatMapWidgetDefinition
	HostMapWidgetDefinition        *HostMapWidgetDefinition
	IFrameWidgetDefinition         *IFrameWidgetDefinition
	ImageWidgetDefinition          *ImageWidgetDefinition
	LogStreamWidgetDefinition      *LogStreamWidgetDefinition
	MonitorSummaryWidgetDefinition *MonitorSummaryWidgetDefinition
	NoteWidgetDefinition           *NoteWidgetDefinition
	QueryValueWidgetDefinition     *QueryValueWidgetDefinition
	ScatterPlotWidgetDefinition    *ScatterPlotWidgetDefinition
	SLOWidgetDefinition            *SLOWidgetDefinition
	ServiceMapWidgetDefinition     *ServiceMapWidgetDefinition
	ServiceSummaryWidgetDefinition *ServiceSummaryWidgetDefinition
	SunburstWidgetDefinition       *SunburstWidgetDefinition
	TableWidgetDefinition          *TableWidgetDefinition
	TimeseriesWidgetDefinition     *TimeseriesWidgetDefinition
	ToplistWidgetDefinition        *ToplistWidgetDefinition
	TreeMapWidgetDefinition        *TreeMapWidgetDefinition
	ListStreamWidgetDefinition     *ListStreamWidgetDefinition
	FunnelWidgetDefinition         *FunnelWidgetDefinition

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// AlertGraphWidgetDefinitionAsWidgetDefinition is a convenience function that returns AlertGraphWidgetDefinition wrapped in WidgetDefinition.
func AlertGraphWidgetDefinitionAsWidgetDefinition(v *AlertGraphWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{AlertGraphWidgetDefinition: v}
}

// AlertValueWidgetDefinitionAsWidgetDefinition is a convenience function that returns AlertValueWidgetDefinition wrapped in WidgetDefinition.
func AlertValueWidgetDefinitionAsWidgetDefinition(v *AlertValueWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{AlertValueWidgetDefinition: v}
}

// ChangeWidgetDefinitionAsWidgetDefinition is a convenience function that returns ChangeWidgetDefinition wrapped in WidgetDefinition.
func ChangeWidgetDefinitionAsWidgetDefinition(v *ChangeWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{ChangeWidgetDefinition: v}
}

// CheckStatusWidgetDefinitionAsWidgetDefinition is a convenience function that returns CheckStatusWidgetDefinition wrapped in WidgetDefinition.
func CheckStatusWidgetDefinitionAsWidgetDefinition(v *CheckStatusWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{CheckStatusWidgetDefinition: v}
}

// DistributionWidgetDefinitionAsWidgetDefinition is a convenience function that returns DistributionWidgetDefinition wrapped in WidgetDefinition.
func DistributionWidgetDefinitionAsWidgetDefinition(v *DistributionWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{DistributionWidgetDefinition: v}
}

// EventStreamWidgetDefinitionAsWidgetDefinition is a convenience function that returns EventStreamWidgetDefinition wrapped in WidgetDefinition.
func EventStreamWidgetDefinitionAsWidgetDefinition(v *EventStreamWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{EventStreamWidgetDefinition: v}
}

// EventTimelineWidgetDefinitionAsWidgetDefinition is a convenience function that returns EventTimelineWidgetDefinition wrapped in WidgetDefinition.
func EventTimelineWidgetDefinitionAsWidgetDefinition(v *EventTimelineWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{EventTimelineWidgetDefinition: v}
}

// FreeTextWidgetDefinitionAsWidgetDefinition is a convenience function that returns FreeTextWidgetDefinition wrapped in WidgetDefinition.
func FreeTextWidgetDefinitionAsWidgetDefinition(v *FreeTextWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{FreeTextWidgetDefinition: v}
}

// GeomapWidgetDefinitionAsWidgetDefinition is a convenience function that returns GeomapWidgetDefinition wrapped in WidgetDefinition.
func GeomapWidgetDefinitionAsWidgetDefinition(v *GeomapWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{GeomapWidgetDefinition: v}
}

// GroupWidgetDefinitionAsWidgetDefinition is a convenience function that returns GroupWidgetDefinition wrapped in WidgetDefinition.
func GroupWidgetDefinitionAsWidgetDefinition(v *GroupWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{GroupWidgetDefinition: v}
}

// HeatMapWidgetDefinitionAsWidgetDefinition is a convenience function that returns HeatMapWidgetDefinition wrapped in WidgetDefinition.
func HeatMapWidgetDefinitionAsWidgetDefinition(v *HeatMapWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{HeatMapWidgetDefinition: v}
}

// HostMapWidgetDefinitionAsWidgetDefinition is a convenience function that returns HostMapWidgetDefinition wrapped in WidgetDefinition.
func HostMapWidgetDefinitionAsWidgetDefinition(v *HostMapWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{HostMapWidgetDefinition: v}
}

// IFrameWidgetDefinitionAsWidgetDefinition is a convenience function that returns IFrameWidgetDefinition wrapped in WidgetDefinition.
func IFrameWidgetDefinitionAsWidgetDefinition(v *IFrameWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{IFrameWidgetDefinition: v}
}

// ImageWidgetDefinitionAsWidgetDefinition is a convenience function that returns ImageWidgetDefinition wrapped in WidgetDefinition.
func ImageWidgetDefinitionAsWidgetDefinition(v *ImageWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{ImageWidgetDefinition: v}
}

// LogStreamWidgetDefinitionAsWidgetDefinition is a convenience function that returns LogStreamWidgetDefinition wrapped in WidgetDefinition.
func LogStreamWidgetDefinitionAsWidgetDefinition(v *LogStreamWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{LogStreamWidgetDefinition: v}
}

// MonitorSummaryWidgetDefinitionAsWidgetDefinition is a convenience function that returns MonitorSummaryWidgetDefinition wrapped in WidgetDefinition.
func MonitorSummaryWidgetDefinitionAsWidgetDefinition(v *MonitorSummaryWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{MonitorSummaryWidgetDefinition: v}
}

// NoteWidgetDefinitionAsWidgetDefinition is a convenience function that returns NoteWidgetDefinition wrapped in WidgetDefinition.
func NoteWidgetDefinitionAsWidgetDefinition(v *NoteWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{NoteWidgetDefinition: v}
}

// QueryValueWidgetDefinitionAsWidgetDefinition is a convenience function that returns QueryValueWidgetDefinition wrapped in WidgetDefinition.
func QueryValueWidgetDefinitionAsWidgetDefinition(v *QueryValueWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{QueryValueWidgetDefinition: v}
}

// ScatterPlotWidgetDefinitionAsWidgetDefinition is a convenience function that returns ScatterPlotWidgetDefinition wrapped in WidgetDefinition.
func ScatterPlotWidgetDefinitionAsWidgetDefinition(v *ScatterPlotWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{ScatterPlotWidgetDefinition: v}
}

// SLOWidgetDefinitionAsWidgetDefinition is a convenience function that returns SLOWidgetDefinition wrapped in WidgetDefinition.
func SLOWidgetDefinitionAsWidgetDefinition(v *SLOWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{SLOWidgetDefinition: v}
}

// ServiceMapWidgetDefinitionAsWidgetDefinition is a convenience function that returns ServiceMapWidgetDefinition wrapped in WidgetDefinition.
func ServiceMapWidgetDefinitionAsWidgetDefinition(v *ServiceMapWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{ServiceMapWidgetDefinition: v}
}

// ServiceSummaryWidgetDefinitionAsWidgetDefinition is a convenience function that returns ServiceSummaryWidgetDefinition wrapped in WidgetDefinition.
func ServiceSummaryWidgetDefinitionAsWidgetDefinition(v *ServiceSummaryWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{ServiceSummaryWidgetDefinition: v}
}

// SunburstWidgetDefinitionAsWidgetDefinition is a convenience function that returns SunburstWidgetDefinition wrapped in WidgetDefinition.
func SunburstWidgetDefinitionAsWidgetDefinition(v *SunburstWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{SunburstWidgetDefinition: v}
}

// TableWidgetDefinitionAsWidgetDefinition is a convenience function that returns TableWidgetDefinition wrapped in WidgetDefinition.
func TableWidgetDefinitionAsWidgetDefinition(v *TableWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{TableWidgetDefinition: v}
}

// TimeseriesWidgetDefinitionAsWidgetDefinition is a convenience function that returns TimeseriesWidgetDefinition wrapped in WidgetDefinition.
func TimeseriesWidgetDefinitionAsWidgetDefinition(v *TimeseriesWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{TimeseriesWidgetDefinition: v}
}

// ToplistWidgetDefinitionAsWidgetDefinition is a convenience function that returns ToplistWidgetDefinition wrapped in WidgetDefinition.
func ToplistWidgetDefinitionAsWidgetDefinition(v *ToplistWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{ToplistWidgetDefinition: v}
}

// TreeMapWidgetDefinitionAsWidgetDefinition is a convenience function that returns TreeMapWidgetDefinition wrapped in WidgetDefinition.
func TreeMapWidgetDefinitionAsWidgetDefinition(v *TreeMapWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{TreeMapWidgetDefinition: v}
}

// ListStreamWidgetDefinitionAsWidgetDefinition is a convenience function that returns ListStreamWidgetDefinition wrapped in WidgetDefinition.
func ListStreamWidgetDefinitionAsWidgetDefinition(v *ListStreamWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{ListStreamWidgetDefinition: v}
}

// FunnelWidgetDefinitionAsWidgetDefinition is a convenience function that returns FunnelWidgetDefinition wrapped in WidgetDefinition.
func FunnelWidgetDefinitionAsWidgetDefinition(v *FunnelWidgetDefinition) WidgetDefinition {
	return WidgetDefinition{FunnelWidgetDefinition: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *WidgetDefinition) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into AlertGraphWidgetDefinition
	err = json.Unmarshal(data, &obj.AlertGraphWidgetDefinition)
	if err == nil {
		if obj.AlertGraphWidgetDefinition != nil && obj.AlertGraphWidgetDefinition.UnparsedObject == nil {
			jsonAlertGraphWidgetDefinition, _ := json.Marshal(obj.AlertGraphWidgetDefinition)
			if string(jsonAlertGraphWidgetDefinition) == "{}" { // empty struct
				obj.AlertGraphWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.AlertGraphWidgetDefinition = nil
		}
	} else {
		obj.AlertGraphWidgetDefinition = nil
	}

	// try to unmarshal data into AlertValueWidgetDefinition
	err = json.Unmarshal(data, &obj.AlertValueWidgetDefinition)
	if err == nil {
		if obj.AlertValueWidgetDefinition != nil && obj.AlertValueWidgetDefinition.UnparsedObject == nil {
			jsonAlertValueWidgetDefinition, _ := json.Marshal(obj.AlertValueWidgetDefinition)
			if string(jsonAlertValueWidgetDefinition) == "{}" { // empty struct
				obj.AlertValueWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.AlertValueWidgetDefinition = nil
		}
	} else {
		obj.AlertValueWidgetDefinition = nil
	}

	// try to unmarshal data into ChangeWidgetDefinition
	err = json.Unmarshal(data, &obj.ChangeWidgetDefinition)
	if err == nil {
		if obj.ChangeWidgetDefinition != nil && obj.ChangeWidgetDefinition.UnparsedObject == nil {
			jsonChangeWidgetDefinition, _ := json.Marshal(obj.ChangeWidgetDefinition)
			if string(jsonChangeWidgetDefinition) == "{}" { // empty struct
				obj.ChangeWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.ChangeWidgetDefinition = nil
		}
	} else {
		obj.ChangeWidgetDefinition = nil
	}

	// try to unmarshal data into CheckStatusWidgetDefinition
	err = json.Unmarshal(data, &obj.CheckStatusWidgetDefinition)
	if err == nil {
		if obj.CheckStatusWidgetDefinition != nil && obj.CheckStatusWidgetDefinition.UnparsedObject == nil {
			jsonCheckStatusWidgetDefinition, _ := json.Marshal(obj.CheckStatusWidgetDefinition)
			if string(jsonCheckStatusWidgetDefinition) == "{}" { // empty struct
				obj.CheckStatusWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.CheckStatusWidgetDefinition = nil
		}
	} else {
		obj.CheckStatusWidgetDefinition = nil
	}

	// try to unmarshal data into DistributionWidgetDefinition
	err = json.Unmarshal(data, &obj.DistributionWidgetDefinition)
	if err == nil {
		if obj.DistributionWidgetDefinition != nil && obj.DistributionWidgetDefinition.UnparsedObject == nil {
			jsonDistributionWidgetDefinition, _ := json.Marshal(obj.DistributionWidgetDefinition)
			if string(jsonDistributionWidgetDefinition) == "{}" { // empty struct
				obj.DistributionWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.DistributionWidgetDefinition = nil
		}
	} else {
		obj.DistributionWidgetDefinition = nil
	}

	// try to unmarshal data into EventStreamWidgetDefinition
	err = json.Unmarshal(data, &obj.EventStreamWidgetDefinition)
	if err == nil {
		if obj.EventStreamWidgetDefinition != nil && obj.EventStreamWidgetDefinition.UnparsedObject == nil {
			jsonEventStreamWidgetDefinition, _ := json.Marshal(obj.EventStreamWidgetDefinition)
			if string(jsonEventStreamWidgetDefinition) == "{}" { // empty struct
				obj.EventStreamWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.EventStreamWidgetDefinition = nil
		}
	} else {
		obj.EventStreamWidgetDefinition = nil
	}

	// try to unmarshal data into EventTimelineWidgetDefinition
	err = json.Unmarshal(data, &obj.EventTimelineWidgetDefinition)
	if err == nil {
		if obj.EventTimelineWidgetDefinition != nil && obj.EventTimelineWidgetDefinition.UnparsedObject == nil {
			jsonEventTimelineWidgetDefinition, _ := json.Marshal(obj.EventTimelineWidgetDefinition)
			if string(jsonEventTimelineWidgetDefinition) == "{}" { // empty struct
				obj.EventTimelineWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.EventTimelineWidgetDefinition = nil
		}
	} else {
		obj.EventTimelineWidgetDefinition = nil
	}

	// try to unmarshal data into FreeTextWidgetDefinition
	err = json.Unmarshal(data, &obj.FreeTextWidgetDefinition)
	if err == nil {
		if obj.FreeTextWidgetDefinition != nil && obj.FreeTextWidgetDefinition.UnparsedObject == nil {
			jsonFreeTextWidgetDefinition, _ := json.Marshal(obj.FreeTextWidgetDefinition)
			if string(jsonFreeTextWidgetDefinition) == "{}" { // empty struct
				obj.FreeTextWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.FreeTextWidgetDefinition = nil
		}
	} else {
		obj.FreeTextWidgetDefinition = nil
	}

	// try to unmarshal data into GeomapWidgetDefinition
	err = json.Unmarshal(data, &obj.GeomapWidgetDefinition)
	if err == nil {
		if obj.GeomapWidgetDefinition != nil && obj.GeomapWidgetDefinition.UnparsedObject == nil {
			jsonGeomapWidgetDefinition, _ := json.Marshal(obj.GeomapWidgetDefinition)
			if string(jsonGeomapWidgetDefinition) == "{}" { // empty struct
				obj.GeomapWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.GeomapWidgetDefinition = nil
		}
	} else {
		obj.GeomapWidgetDefinition = nil
	}

	// try to unmarshal data into GroupWidgetDefinition
	err = json.Unmarshal(data, &obj.GroupWidgetDefinition)
	if err == nil {
		if obj.GroupWidgetDefinition != nil && obj.GroupWidgetDefinition.UnparsedObject == nil {
			jsonGroupWidgetDefinition, _ := json.Marshal(obj.GroupWidgetDefinition)
			if string(jsonGroupWidgetDefinition) == "{}" { // empty struct
				obj.GroupWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.GroupWidgetDefinition = nil
		}
	} else {
		obj.GroupWidgetDefinition = nil
	}

	// try to unmarshal data into HeatMapWidgetDefinition
	err = json.Unmarshal(data, &obj.HeatMapWidgetDefinition)
	if err == nil {
		if obj.HeatMapWidgetDefinition != nil && obj.HeatMapWidgetDefinition.UnparsedObject == nil {
			jsonHeatMapWidgetDefinition, _ := json.Marshal(obj.HeatMapWidgetDefinition)
			if string(jsonHeatMapWidgetDefinition) == "{}" { // empty struct
				obj.HeatMapWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.HeatMapWidgetDefinition = nil
		}
	} else {
		obj.HeatMapWidgetDefinition = nil
	}

	// try to unmarshal data into HostMapWidgetDefinition
	err = json.Unmarshal(data, &obj.HostMapWidgetDefinition)
	if err == nil {
		if obj.HostMapWidgetDefinition != nil && obj.HostMapWidgetDefinition.UnparsedObject == nil {
			jsonHostMapWidgetDefinition, _ := json.Marshal(obj.HostMapWidgetDefinition)
			if string(jsonHostMapWidgetDefinition) == "{}" { // empty struct
				obj.HostMapWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.HostMapWidgetDefinition = nil
		}
	} else {
		obj.HostMapWidgetDefinition = nil
	}

	// try to unmarshal data into IFrameWidgetDefinition
	err = json.Unmarshal(data, &obj.IFrameWidgetDefinition)
	if err == nil {
		if obj.IFrameWidgetDefinition != nil && obj.IFrameWidgetDefinition.UnparsedObject == nil {
			jsonIFrameWidgetDefinition, _ := json.Marshal(obj.IFrameWidgetDefinition)
			if string(jsonIFrameWidgetDefinition) == "{}" { // empty struct
				obj.IFrameWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.IFrameWidgetDefinition = nil
		}
	} else {
		obj.IFrameWidgetDefinition = nil
	}

	// try to unmarshal data into ImageWidgetDefinition
	err = json.Unmarshal(data, &obj.ImageWidgetDefinition)
	if err == nil {
		if obj.ImageWidgetDefinition != nil && obj.ImageWidgetDefinition.UnparsedObject == nil {
			jsonImageWidgetDefinition, _ := json.Marshal(obj.ImageWidgetDefinition)
			if string(jsonImageWidgetDefinition) == "{}" { // empty struct
				obj.ImageWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.ImageWidgetDefinition = nil
		}
	} else {
		obj.ImageWidgetDefinition = nil
	}

	// try to unmarshal data into LogStreamWidgetDefinition
	err = json.Unmarshal(data, &obj.LogStreamWidgetDefinition)
	if err == nil {
		if obj.LogStreamWidgetDefinition != nil && obj.LogStreamWidgetDefinition.UnparsedObject == nil {
			jsonLogStreamWidgetDefinition, _ := json.Marshal(obj.LogStreamWidgetDefinition)
			if string(jsonLogStreamWidgetDefinition) == "{}" { // empty struct
				obj.LogStreamWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.LogStreamWidgetDefinition = nil
		}
	} else {
		obj.LogStreamWidgetDefinition = nil
	}

	// try to unmarshal data into MonitorSummaryWidgetDefinition
	err = json.Unmarshal(data, &obj.MonitorSummaryWidgetDefinition)
	if err == nil {
		if obj.MonitorSummaryWidgetDefinition != nil && obj.MonitorSummaryWidgetDefinition.UnparsedObject == nil {
			jsonMonitorSummaryWidgetDefinition, _ := json.Marshal(obj.MonitorSummaryWidgetDefinition)
			if string(jsonMonitorSummaryWidgetDefinition) == "{}" { // empty struct
				obj.MonitorSummaryWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.MonitorSummaryWidgetDefinition = nil
		}
	} else {
		obj.MonitorSummaryWidgetDefinition = nil
	}

	// try to unmarshal data into NoteWidgetDefinition
	err = json.Unmarshal(data, &obj.NoteWidgetDefinition)
	if err == nil {
		if obj.NoteWidgetDefinition != nil && obj.NoteWidgetDefinition.UnparsedObject == nil {
			jsonNoteWidgetDefinition, _ := json.Marshal(obj.NoteWidgetDefinition)
			if string(jsonNoteWidgetDefinition) == "{}" { // empty struct
				obj.NoteWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.NoteWidgetDefinition = nil
		}
	} else {
		obj.NoteWidgetDefinition = nil
	}

	// try to unmarshal data into QueryValueWidgetDefinition
	err = json.Unmarshal(data, &obj.QueryValueWidgetDefinition)
	if err == nil {
		if obj.QueryValueWidgetDefinition != nil && obj.QueryValueWidgetDefinition.UnparsedObject == nil {
			jsonQueryValueWidgetDefinition, _ := json.Marshal(obj.QueryValueWidgetDefinition)
			if string(jsonQueryValueWidgetDefinition) == "{}" { // empty struct
				obj.QueryValueWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.QueryValueWidgetDefinition = nil
		}
	} else {
		obj.QueryValueWidgetDefinition = nil
	}

	// try to unmarshal data into ScatterPlotWidgetDefinition
	err = json.Unmarshal(data, &obj.ScatterPlotWidgetDefinition)
	if err == nil {
		if obj.ScatterPlotWidgetDefinition != nil && obj.ScatterPlotWidgetDefinition.UnparsedObject == nil {
			jsonScatterPlotWidgetDefinition, _ := json.Marshal(obj.ScatterPlotWidgetDefinition)
			if string(jsonScatterPlotWidgetDefinition) == "{}" { // empty struct
				obj.ScatterPlotWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.ScatterPlotWidgetDefinition = nil
		}
	} else {
		obj.ScatterPlotWidgetDefinition = nil
	}

	// try to unmarshal data into SLOWidgetDefinition
	err = json.Unmarshal(data, &obj.SLOWidgetDefinition)
	if err == nil {
		if obj.SLOWidgetDefinition != nil && obj.SLOWidgetDefinition.UnparsedObject == nil {
			jsonSLOWidgetDefinition, _ := json.Marshal(obj.SLOWidgetDefinition)
			if string(jsonSLOWidgetDefinition) == "{}" { // empty struct
				obj.SLOWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.SLOWidgetDefinition = nil
		}
	} else {
		obj.SLOWidgetDefinition = nil
	}

	// try to unmarshal data into ServiceMapWidgetDefinition
	err = json.Unmarshal(data, &obj.ServiceMapWidgetDefinition)
	if err == nil {
		if obj.ServiceMapWidgetDefinition != nil && obj.ServiceMapWidgetDefinition.UnparsedObject == nil {
			jsonServiceMapWidgetDefinition, _ := json.Marshal(obj.ServiceMapWidgetDefinition)
			if string(jsonServiceMapWidgetDefinition) == "{}" { // empty struct
				obj.ServiceMapWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.ServiceMapWidgetDefinition = nil
		}
	} else {
		obj.ServiceMapWidgetDefinition = nil
	}

	// try to unmarshal data into ServiceSummaryWidgetDefinition
	err = json.Unmarshal(data, &obj.ServiceSummaryWidgetDefinition)
	if err == nil {
		if obj.ServiceSummaryWidgetDefinition != nil && obj.ServiceSummaryWidgetDefinition.UnparsedObject == nil {
			jsonServiceSummaryWidgetDefinition, _ := json.Marshal(obj.ServiceSummaryWidgetDefinition)
			if string(jsonServiceSummaryWidgetDefinition) == "{}" { // empty struct
				obj.ServiceSummaryWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.ServiceSummaryWidgetDefinition = nil
		}
	} else {
		obj.ServiceSummaryWidgetDefinition = nil
	}

	// try to unmarshal data into SunburstWidgetDefinition
	err = json.Unmarshal(data, &obj.SunburstWidgetDefinition)
	if err == nil {
		if obj.SunburstWidgetDefinition != nil && obj.SunburstWidgetDefinition.UnparsedObject == nil {
			jsonSunburstWidgetDefinition, _ := json.Marshal(obj.SunburstWidgetDefinition)
			if string(jsonSunburstWidgetDefinition) == "{}" { // empty struct
				obj.SunburstWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.SunburstWidgetDefinition = nil
		}
	} else {
		obj.SunburstWidgetDefinition = nil
	}

	// try to unmarshal data into TableWidgetDefinition
	err = json.Unmarshal(data, &obj.TableWidgetDefinition)
	if err == nil {
		if obj.TableWidgetDefinition != nil && obj.TableWidgetDefinition.UnparsedObject == nil {
			jsonTableWidgetDefinition, _ := json.Marshal(obj.TableWidgetDefinition)
			if string(jsonTableWidgetDefinition) == "{}" { // empty struct
				obj.TableWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.TableWidgetDefinition = nil
		}
	} else {
		obj.TableWidgetDefinition = nil
	}

	// try to unmarshal data into TimeseriesWidgetDefinition
	err = json.Unmarshal(data, &obj.TimeseriesWidgetDefinition)
	if err == nil {
		if obj.TimeseriesWidgetDefinition != nil && obj.TimeseriesWidgetDefinition.UnparsedObject == nil {
			jsonTimeseriesWidgetDefinition, _ := json.Marshal(obj.TimeseriesWidgetDefinition)
			if string(jsonTimeseriesWidgetDefinition) == "{}" { // empty struct
				obj.TimeseriesWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.TimeseriesWidgetDefinition = nil
		}
	} else {
		obj.TimeseriesWidgetDefinition = nil
	}

	// try to unmarshal data into ToplistWidgetDefinition
	err = json.Unmarshal(data, &obj.ToplistWidgetDefinition)
	if err == nil {
		if obj.ToplistWidgetDefinition != nil && obj.ToplistWidgetDefinition.UnparsedObject == nil {
			jsonToplistWidgetDefinition, _ := json.Marshal(obj.ToplistWidgetDefinition)
			if string(jsonToplistWidgetDefinition) == "{}" { // empty struct
				obj.ToplistWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.ToplistWidgetDefinition = nil
		}
	} else {
		obj.ToplistWidgetDefinition = nil
	}

	// try to unmarshal data into TreeMapWidgetDefinition
	err = json.Unmarshal(data, &obj.TreeMapWidgetDefinition)
	if err == nil {
		if obj.TreeMapWidgetDefinition != nil && obj.TreeMapWidgetDefinition.UnparsedObject == nil {
			jsonTreeMapWidgetDefinition, _ := json.Marshal(obj.TreeMapWidgetDefinition)
			if string(jsonTreeMapWidgetDefinition) == "{}" { // empty struct
				obj.TreeMapWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.TreeMapWidgetDefinition = nil
		}
	} else {
		obj.TreeMapWidgetDefinition = nil
	}

	// try to unmarshal data into ListStreamWidgetDefinition
	err = json.Unmarshal(data, &obj.ListStreamWidgetDefinition)
	if err == nil {
		if obj.ListStreamWidgetDefinition != nil && obj.ListStreamWidgetDefinition.UnparsedObject == nil {
			jsonListStreamWidgetDefinition, _ := json.Marshal(obj.ListStreamWidgetDefinition)
			if string(jsonListStreamWidgetDefinition) == "{}" { // empty struct
				obj.ListStreamWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.ListStreamWidgetDefinition = nil
		}
	} else {
		obj.ListStreamWidgetDefinition = nil
	}

	// try to unmarshal data into FunnelWidgetDefinition
	err = json.Unmarshal(data, &obj.FunnelWidgetDefinition)
	if err == nil {
		if obj.FunnelWidgetDefinition != nil && obj.FunnelWidgetDefinition.UnparsedObject == nil {
			jsonFunnelWidgetDefinition, _ := json.Marshal(obj.FunnelWidgetDefinition)
			if string(jsonFunnelWidgetDefinition) == "{}" { // empty struct
				obj.FunnelWidgetDefinition = nil
			} else {
				match++
			}
		} else {
			obj.FunnelWidgetDefinition = nil
		}
	} else {
		obj.FunnelWidgetDefinition = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.AlertGraphWidgetDefinition = nil
		obj.AlertValueWidgetDefinition = nil
		obj.ChangeWidgetDefinition = nil
		obj.CheckStatusWidgetDefinition = nil
		obj.DistributionWidgetDefinition = nil
		obj.EventStreamWidgetDefinition = nil
		obj.EventTimelineWidgetDefinition = nil
		obj.FreeTextWidgetDefinition = nil
		obj.GeomapWidgetDefinition = nil
		obj.GroupWidgetDefinition = nil
		obj.HeatMapWidgetDefinition = nil
		obj.HostMapWidgetDefinition = nil
		obj.IFrameWidgetDefinition = nil
		obj.ImageWidgetDefinition = nil
		obj.LogStreamWidgetDefinition = nil
		obj.MonitorSummaryWidgetDefinition = nil
		obj.NoteWidgetDefinition = nil
		obj.QueryValueWidgetDefinition = nil
		obj.ScatterPlotWidgetDefinition = nil
		obj.SLOWidgetDefinition = nil
		obj.ServiceMapWidgetDefinition = nil
		obj.ServiceSummaryWidgetDefinition = nil
		obj.SunburstWidgetDefinition = nil
		obj.TableWidgetDefinition = nil
		obj.TimeseriesWidgetDefinition = nil
		obj.ToplistWidgetDefinition = nil
		obj.TreeMapWidgetDefinition = nil
		obj.ListStreamWidgetDefinition = nil
		obj.FunnelWidgetDefinition = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj WidgetDefinition) MarshalJSON() ([]byte, error) {
	if obj.AlertGraphWidgetDefinition != nil {
		return json.Marshal(&obj.AlertGraphWidgetDefinition)
	}

	if obj.AlertValueWidgetDefinition != nil {
		return json.Marshal(&obj.AlertValueWidgetDefinition)
	}

	if obj.ChangeWidgetDefinition != nil {
		return json.Marshal(&obj.ChangeWidgetDefinition)
	}

	if obj.CheckStatusWidgetDefinition != nil {
		return json.Marshal(&obj.CheckStatusWidgetDefinition)
	}

	if obj.DistributionWidgetDefinition != nil {
		return json.Marshal(&obj.DistributionWidgetDefinition)
	}

	if obj.EventStreamWidgetDefinition != nil {
		return json.Marshal(&obj.EventStreamWidgetDefinition)
	}

	if obj.EventTimelineWidgetDefinition != nil {
		return json.Marshal(&obj.EventTimelineWidgetDefinition)
	}

	if obj.FreeTextWidgetDefinition != nil {
		return json.Marshal(&obj.FreeTextWidgetDefinition)
	}

	if obj.GeomapWidgetDefinition != nil {
		return json.Marshal(&obj.GeomapWidgetDefinition)
	}

	if obj.GroupWidgetDefinition != nil {
		return json.Marshal(&obj.GroupWidgetDefinition)
	}

	if obj.HeatMapWidgetDefinition != nil {
		return json.Marshal(&obj.HeatMapWidgetDefinition)
	}

	if obj.HostMapWidgetDefinition != nil {
		return json.Marshal(&obj.HostMapWidgetDefinition)
	}

	if obj.IFrameWidgetDefinition != nil {
		return json.Marshal(&obj.IFrameWidgetDefinition)
	}

	if obj.ImageWidgetDefinition != nil {
		return json.Marshal(&obj.ImageWidgetDefinition)
	}

	if obj.LogStreamWidgetDefinition != nil {
		return json.Marshal(&obj.LogStreamWidgetDefinition)
	}

	if obj.MonitorSummaryWidgetDefinition != nil {
		return json.Marshal(&obj.MonitorSummaryWidgetDefinition)
	}

	if obj.NoteWidgetDefinition != nil {
		return json.Marshal(&obj.NoteWidgetDefinition)
	}

	if obj.QueryValueWidgetDefinition != nil {
		return json.Marshal(&obj.QueryValueWidgetDefinition)
	}

	if obj.ScatterPlotWidgetDefinition != nil {
		return json.Marshal(&obj.ScatterPlotWidgetDefinition)
	}

	if obj.SLOWidgetDefinition != nil {
		return json.Marshal(&obj.SLOWidgetDefinition)
	}

	if obj.ServiceMapWidgetDefinition != nil {
		return json.Marshal(&obj.ServiceMapWidgetDefinition)
	}

	if obj.ServiceSummaryWidgetDefinition != nil {
		return json.Marshal(&obj.ServiceSummaryWidgetDefinition)
	}

	if obj.SunburstWidgetDefinition != nil {
		return json.Marshal(&obj.SunburstWidgetDefinition)
	}

	if obj.TableWidgetDefinition != nil {
		return json.Marshal(&obj.TableWidgetDefinition)
	}

	if obj.TimeseriesWidgetDefinition != nil {
		return json.Marshal(&obj.TimeseriesWidgetDefinition)
	}

	if obj.ToplistWidgetDefinition != nil {
		return json.Marshal(&obj.ToplistWidgetDefinition)
	}

	if obj.TreeMapWidgetDefinition != nil {
		return json.Marshal(&obj.TreeMapWidgetDefinition)
	}

	if obj.ListStreamWidgetDefinition != nil {
		return json.Marshal(&obj.ListStreamWidgetDefinition)
	}

	if obj.FunnelWidgetDefinition != nil {
		return json.Marshal(&obj.FunnelWidgetDefinition)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *WidgetDefinition) GetActualInstance() interface{} {
	if obj.AlertGraphWidgetDefinition != nil {
		return obj.AlertGraphWidgetDefinition
	}

	if obj.AlertValueWidgetDefinition != nil {
		return obj.AlertValueWidgetDefinition
	}

	if obj.ChangeWidgetDefinition != nil {
		return obj.ChangeWidgetDefinition
	}

	if obj.CheckStatusWidgetDefinition != nil {
		return obj.CheckStatusWidgetDefinition
	}

	if obj.DistributionWidgetDefinition != nil {
		return obj.DistributionWidgetDefinition
	}

	if obj.EventStreamWidgetDefinition != nil {
		return obj.EventStreamWidgetDefinition
	}

	if obj.EventTimelineWidgetDefinition != nil {
		return obj.EventTimelineWidgetDefinition
	}

	if obj.FreeTextWidgetDefinition != nil {
		return obj.FreeTextWidgetDefinition
	}

	if obj.GeomapWidgetDefinition != nil {
		return obj.GeomapWidgetDefinition
	}

	if obj.GroupWidgetDefinition != nil {
		return obj.GroupWidgetDefinition
	}

	if obj.HeatMapWidgetDefinition != nil {
		return obj.HeatMapWidgetDefinition
	}

	if obj.HostMapWidgetDefinition != nil {
		return obj.HostMapWidgetDefinition
	}

	if obj.IFrameWidgetDefinition != nil {
		return obj.IFrameWidgetDefinition
	}

	if obj.ImageWidgetDefinition != nil {
		return obj.ImageWidgetDefinition
	}

	if obj.LogStreamWidgetDefinition != nil {
		return obj.LogStreamWidgetDefinition
	}

	if obj.MonitorSummaryWidgetDefinition != nil {
		return obj.MonitorSummaryWidgetDefinition
	}

	if obj.NoteWidgetDefinition != nil {
		return obj.NoteWidgetDefinition
	}

	if obj.QueryValueWidgetDefinition != nil {
		return obj.QueryValueWidgetDefinition
	}

	if obj.ScatterPlotWidgetDefinition != nil {
		return obj.ScatterPlotWidgetDefinition
	}

	if obj.SLOWidgetDefinition != nil {
		return obj.SLOWidgetDefinition
	}

	if obj.ServiceMapWidgetDefinition != nil {
		return obj.ServiceMapWidgetDefinition
	}

	if obj.ServiceSummaryWidgetDefinition != nil {
		return obj.ServiceSummaryWidgetDefinition
	}

	if obj.SunburstWidgetDefinition != nil {
		return obj.SunburstWidgetDefinition
	}

	if obj.TableWidgetDefinition != nil {
		return obj.TableWidgetDefinition
	}

	if obj.TimeseriesWidgetDefinition != nil {
		return obj.TimeseriesWidgetDefinition
	}

	if obj.ToplistWidgetDefinition != nil {
		return obj.ToplistWidgetDefinition
	}

	if obj.TreeMapWidgetDefinition != nil {
		return obj.TreeMapWidgetDefinition
	}

	if obj.ListStreamWidgetDefinition != nil {
		return obj.ListStreamWidgetDefinition
	}

	if obj.FunnelWidgetDefinition != nil {
		return obj.FunnelWidgetDefinition
	}

	// all schemas are nil
	return nil
}

// NullableWidgetDefinition handles when a null is used for WidgetDefinition.
type NullableWidgetDefinition struct {
	value *WidgetDefinition
	isSet bool
}

// Get returns the associated value.
func (v NullableWidgetDefinition) Get() *WidgetDefinition {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableWidgetDefinition) Set(val *WidgetDefinition) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableWidgetDefinition) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableWidgetDefinition) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableWidgetDefinition initializes the struct as if Set has been called.
func NewNullableWidgetDefinition(val *WidgetDefinition) *NullableWidgetDefinition {
	return &NullableWidgetDefinition{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableWidgetDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableWidgetDefinition) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
