// Unless explicitly stated otherwise all files in this repository are licensed under the Apache-2.0 License.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2019-Present Datadog, Inc.

package datadog

import (
	"encoding/json"
)

// LogsProcessor - Definition of a logs processor.
type LogsProcessor struct {
	LogsGrokParser             *LogsGrokParser
	LogsDateRemapper           *LogsDateRemapper
	LogsStatusRemapper         *LogsStatusRemapper
	LogsServiceRemapper        *LogsServiceRemapper
	LogsMessageRemapper        *LogsMessageRemapper
	LogsAttributeRemapper      *LogsAttributeRemapper
	LogsURLParser              *LogsURLParser
	LogsUserAgentParser        *LogsUserAgentParser
	LogsCategoryProcessor      *LogsCategoryProcessor
	LogsArithmeticProcessor    *LogsArithmeticProcessor
	LogsStringBuilderProcessor *LogsStringBuilderProcessor
	LogsPipelineProcessor      *LogsPipelineProcessor
	LogsGeoIPParser            *LogsGeoIPParser
	LogsLookupProcessor        *LogsLookupProcessor
	LogsTraceRemapper          *LogsTraceRemapper

	// UnparsedObject contains the raw value of the object if there was an error when deserializing into the struct
	UnparsedObject interface{}
}

// LogsGrokParserAsLogsProcessor is a convenience function that returns LogsGrokParser wrapped in LogsProcessor.
func LogsGrokParserAsLogsProcessor(v *LogsGrokParser) LogsProcessor {
	return LogsProcessor{LogsGrokParser: v}
}

// LogsDateRemapperAsLogsProcessor is a convenience function that returns LogsDateRemapper wrapped in LogsProcessor.
func LogsDateRemapperAsLogsProcessor(v *LogsDateRemapper) LogsProcessor {
	return LogsProcessor{LogsDateRemapper: v}
}

// LogsStatusRemapperAsLogsProcessor is a convenience function that returns LogsStatusRemapper wrapped in LogsProcessor.
func LogsStatusRemapperAsLogsProcessor(v *LogsStatusRemapper) LogsProcessor {
	return LogsProcessor{LogsStatusRemapper: v}
}

// LogsServiceRemapperAsLogsProcessor is a convenience function that returns LogsServiceRemapper wrapped in LogsProcessor.
func LogsServiceRemapperAsLogsProcessor(v *LogsServiceRemapper) LogsProcessor {
	return LogsProcessor{LogsServiceRemapper: v}
}

// LogsMessageRemapperAsLogsProcessor is a convenience function that returns LogsMessageRemapper wrapped in LogsProcessor.
func LogsMessageRemapperAsLogsProcessor(v *LogsMessageRemapper) LogsProcessor {
	return LogsProcessor{LogsMessageRemapper: v}
}

// LogsAttributeRemapperAsLogsProcessor is a convenience function that returns LogsAttributeRemapper wrapped in LogsProcessor.
func LogsAttributeRemapperAsLogsProcessor(v *LogsAttributeRemapper) LogsProcessor {
	return LogsProcessor{LogsAttributeRemapper: v}
}

// LogsURLParserAsLogsProcessor is a convenience function that returns LogsURLParser wrapped in LogsProcessor.
func LogsURLParserAsLogsProcessor(v *LogsURLParser) LogsProcessor {
	return LogsProcessor{LogsURLParser: v}
}

// LogsUserAgentParserAsLogsProcessor is a convenience function that returns LogsUserAgentParser wrapped in LogsProcessor.
func LogsUserAgentParserAsLogsProcessor(v *LogsUserAgentParser) LogsProcessor {
	return LogsProcessor{LogsUserAgentParser: v}
}

// LogsCategoryProcessorAsLogsProcessor is a convenience function that returns LogsCategoryProcessor wrapped in LogsProcessor.
func LogsCategoryProcessorAsLogsProcessor(v *LogsCategoryProcessor) LogsProcessor {
	return LogsProcessor{LogsCategoryProcessor: v}
}

// LogsArithmeticProcessorAsLogsProcessor is a convenience function that returns LogsArithmeticProcessor wrapped in LogsProcessor.
func LogsArithmeticProcessorAsLogsProcessor(v *LogsArithmeticProcessor) LogsProcessor {
	return LogsProcessor{LogsArithmeticProcessor: v}
}

// LogsStringBuilderProcessorAsLogsProcessor is a convenience function that returns LogsStringBuilderProcessor wrapped in LogsProcessor.
func LogsStringBuilderProcessorAsLogsProcessor(v *LogsStringBuilderProcessor) LogsProcessor {
	return LogsProcessor{LogsStringBuilderProcessor: v}
}

// LogsPipelineProcessorAsLogsProcessor is a convenience function that returns LogsPipelineProcessor wrapped in LogsProcessor.
func LogsPipelineProcessorAsLogsProcessor(v *LogsPipelineProcessor) LogsProcessor {
	return LogsProcessor{LogsPipelineProcessor: v}
}

// LogsGeoIPParserAsLogsProcessor is a convenience function that returns LogsGeoIPParser wrapped in LogsProcessor.
func LogsGeoIPParserAsLogsProcessor(v *LogsGeoIPParser) LogsProcessor {
	return LogsProcessor{LogsGeoIPParser: v}
}

// LogsLookupProcessorAsLogsProcessor is a convenience function that returns LogsLookupProcessor wrapped in LogsProcessor.
func LogsLookupProcessorAsLogsProcessor(v *LogsLookupProcessor) LogsProcessor {
	return LogsProcessor{LogsLookupProcessor: v}
}

// LogsTraceRemapperAsLogsProcessor is a convenience function that returns LogsTraceRemapper wrapped in LogsProcessor.
func LogsTraceRemapperAsLogsProcessor(v *LogsTraceRemapper) LogsProcessor {
	return LogsProcessor{LogsTraceRemapper: v}
}

// UnmarshalJSON turns data into one of the pointers in the struct.
func (obj *LogsProcessor) UnmarshalJSON(data []byte) error {
	var err error
	match := 0
	// try to unmarshal data into LogsGrokParser
	err = json.Unmarshal(data, &obj.LogsGrokParser)
	if err == nil {
		if obj.LogsGrokParser != nil && obj.LogsGrokParser.UnparsedObject == nil {
			jsonLogsGrokParser, _ := json.Marshal(obj.LogsGrokParser)
			if string(jsonLogsGrokParser) == "{}" { // empty struct
				obj.LogsGrokParser = nil
			} else {
				match++
			}
		} else {
			obj.LogsGrokParser = nil
		}
	} else {
		obj.LogsGrokParser = nil
	}

	// try to unmarshal data into LogsDateRemapper
	err = json.Unmarshal(data, &obj.LogsDateRemapper)
	if err == nil {
		if obj.LogsDateRemapper != nil && obj.LogsDateRemapper.UnparsedObject == nil {
			jsonLogsDateRemapper, _ := json.Marshal(obj.LogsDateRemapper)
			if string(jsonLogsDateRemapper) == "{}" { // empty struct
				obj.LogsDateRemapper = nil
			} else {
				match++
			}
		} else {
			obj.LogsDateRemapper = nil
		}
	} else {
		obj.LogsDateRemapper = nil
	}

	// try to unmarshal data into LogsStatusRemapper
	err = json.Unmarshal(data, &obj.LogsStatusRemapper)
	if err == nil {
		if obj.LogsStatusRemapper != nil && obj.LogsStatusRemapper.UnparsedObject == nil {
			jsonLogsStatusRemapper, _ := json.Marshal(obj.LogsStatusRemapper)
			if string(jsonLogsStatusRemapper) == "{}" { // empty struct
				obj.LogsStatusRemapper = nil
			} else {
				match++
			}
		} else {
			obj.LogsStatusRemapper = nil
		}
	} else {
		obj.LogsStatusRemapper = nil
	}

	// try to unmarshal data into LogsServiceRemapper
	err = json.Unmarshal(data, &obj.LogsServiceRemapper)
	if err == nil {
		if obj.LogsServiceRemapper != nil && obj.LogsServiceRemapper.UnparsedObject == nil {
			jsonLogsServiceRemapper, _ := json.Marshal(obj.LogsServiceRemapper)
			if string(jsonLogsServiceRemapper) == "{}" { // empty struct
				obj.LogsServiceRemapper = nil
			} else {
				match++
			}
		} else {
			obj.LogsServiceRemapper = nil
		}
	} else {
		obj.LogsServiceRemapper = nil
	}

	// try to unmarshal data into LogsMessageRemapper
	err = json.Unmarshal(data, &obj.LogsMessageRemapper)
	if err == nil {
		if obj.LogsMessageRemapper != nil && obj.LogsMessageRemapper.UnparsedObject == nil {
			jsonLogsMessageRemapper, _ := json.Marshal(obj.LogsMessageRemapper)
			if string(jsonLogsMessageRemapper) == "{}" { // empty struct
				obj.LogsMessageRemapper = nil
			} else {
				match++
			}
		} else {
			obj.LogsMessageRemapper = nil
		}
	} else {
		obj.LogsMessageRemapper = nil
	}

	// try to unmarshal data into LogsAttributeRemapper
	err = json.Unmarshal(data, &obj.LogsAttributeRemapper)
	if err == nil {
		if obj.LogsAttributeRemapper != nil && obj.LogsAttributeRemapper.UnparsedObject == nil {
			jsonLogsAttributeRemapper, _ := json.Marshal(obj.LogsAttributeRemapper)
			if string(jsonLogsAttributeRemapper) == "{}" { // empty struct
				obj.LogsAttributeRemapper = nil
			} else {
				match++
			}
		} else {
			obj.LogsAttributeRemapper = nil
		}
	} else {
		obj.LogsAttributeRemapper = nil
	}

	// try to unmarshal data into LogsURLParser
	err = json.Unmarshal(data, &obj.LogsURLParser)
	if err == nil {
		if obj.LogsURLParser != nil && obj.LogsURLParser.UnparsedObject == nil {
			jsonLogsURLParser, _ := json.Marshal(obj.LogsURLParser)
			if string(jsonLogsURLParser) == "{}" { // empty struct
				obj.LogsURLParser = nil
			} else {
				match++
			}
		} else {
			obj.LogsURLParser = nil
		}
	} else {
		obj.LogsURLParser = nil
	}

	// try to unmarshal data into LogsUserAgentParser
	err = json.Unmarshal(data, &obj.LogsUserAgentParser)
	if err == nil {
		if obj.LogsUserAgentParser != nil && obj.LogsUserAgentParser.UnparsedObject == nil {
			jsonLogsUserAgentParser, _ := json.Marshal(obj.LogsUserAgentParser)
			if string(jsonLogsUserAgentParser) == "{}" { // empty struct
				obj.LogsUserAgentParser = nil
			} else {
				match++
			}
		} else {
			obj.LogsUserAgentParser = nil
		}
	} else {
		obj.LogsUserAgentParser = nil
	}

	// try to unmarshal data into LogsCategoryProcessor
	err = json.Unmarshal(data, &obj.LogsCategoryProcessor)
	if err == nil {
		if obj.LogsCategoryProcessor != nil && obj.LogsCategoryProcessor.UnparsedObject == nil {
			jsonLogsCategoryProcessor, _ := json.Marshal(obj.LogsCategoryProcessor)
			if string(jsonLogsCategoryProcessor) == "{}" { // empty struct
				obj.LogsCategoryProcessor = nil
			} else {
				match++
			}
		} else {
			obj.LogsCategoryProcessor = nil
		}
	} else {
		obj.LogsCategoryProcessor = nil
	}

	// try to unmarshal data into LogsArithmeticProcessor
	err = json.Unmarshal(data, &obj.LogsArithmeticProcessor)
	if err == nil {
		if obj.LogsArithmeticProcessor != nil && obj.LogsArithmeticProcessor.UnparsedObject == nil {
			jsonLogsArithmeticProcessor, _ := json.Marshal(obj.LogsArithmeticProcessor)
			if string(jsonLogsArithmeticProcessor) == "{}" { // empty struct
				obj.LogsArithmeticProcessor = nil
			} else {
				match++
			}
		} else {
			obj.LogsArithmeticProcessor = nil
		}
	} else {
		obj.LogsArithmeticProcessor = nil
	}

	// try to unmarshal data into LogsStringBuilderProcessor
	err = json.Unmarshal(data, &obj.LogsStringBuilderProcessor)
	if err == nil {
		if obj.LogsStringBuilderProcessor != nil && obj.LogsStringBuilderProcessor.UnparsedObject == nil {
			jsonLogsStringBuilderProcessor, _ := json.Marshal(obj.LogsStringBuilderProcessor)
			if string(jsonLogsStringBuilderProcessor) == "{}" { // empty struct
				obj.LogsStringBuilderProcessor = nil
			} else {
				match++
			}
		} else {
			obj.LogsStringBuilderProcessor = nil
		}
	} else {
		obj.LogsStringBuilderProcessor = nil
	}

	// try to unmarshal data into LogsPipelineProcessor
	err = json.Unmarshal(data, &obj.LogsPipelineProcessor)
	if err == nil {
		if obj.LogsPipelineProcessor != nil && obj.LogsPipelineProcessor.UnparsedObject == nil {
			jsonLogsPipelineProcessor, _ := json.Marshal(obj.LogsPipelineProcessor)
			if string(jsonLogsPipelineProcessor) == "{}" { // empty struct
				obj.LogsPipelineProcessor = nil
			} else {
				match++
			}
		} else {
			obj.LogsPipelineProcessor = nil
		}
	} else {
		obj.LogsPipelineProcessor = nil
	}

	// try to unmarshal data into LogsGeoIPParser
	err = json.Unmarshal(data, &obj.LogsGeoIPParser)
	if err == nil {
		if obj.LogsGeoIPParser != nil && obj.LogsGeoIPParser.UnparsedObject == nil {
			jsonLogsGeoIPParser, _ := json.Marshal(obj.LogsGeoIPParser)
			if string(jsonLogsGeoIPParser) == "{}" { // empty struct
				obj.LogsGeoIPParser = nil
			} else {
				match++
			}
		} else {
			obj.LogsGeoIPParser = nil
		}
	} else {
		obj.LogsGeoIPParser = nil
	}

	// try to unmarshal data into LogsLookupProcessor
	err = json.Unmarshal(data, &obj.LogsLookupProcessor)
	if err == nil {
		if obj.LogsLookupProcessor != nil && obj.LogsLookupProcessor.UnparsedObject == nil {
			jsonLogsLookupProcessor, _ := json.Marshal(obj.LogsLookupProcessor)
			if string(jsonLogsLookupProcessor) == "{}" { // empty struct
				obj.LogsLookupProcessor = nil
			} else {
				match++
			}
		} else {
			obj.LogsLookupProcessor = nil
		}
	} else {
		obj.LogsLookupProcessor = nil
	}

	// try to unmarshal data into LogsTraceRemapper
	err = json.Unmarshal(data, &obj.LogsTraceRemapper)
	if err == nil {
		if obj.LogsTraceRemapper != nil && obj.LogsTraceRemapper.UnparsedObject == nil {
			jsonLogsTraceRemapper, _ := json.Marshal(obj.LogsTraceRemapper)
			if string(jsonLogsTraceRemapper) == "{}" { // empty struct
				obj.LogsTraceRemapper = nil
			} else {
				match++
			}
		} else {
			obj.LogsTraceRemapper = nil
		}
	} else {
		obj.LogsTraceRemapper = nil
	}

	if match != 1 { // more than 1 match
		// reset to nil
		obj.LogsGrokParser = nil
		obj.LogsDateRemapper = nil
		obj.LogsStatusRemapper = nil
		obj.LogsServiceRemapper = nil
		obj.LogsMessageRemapper = nil
		obj.LogsAttributeRemapper = nil
		obj.LogsURLParser = nil
		obj.LogsUserAgentParser = nil
		obj.LogsCategoryProcessor = nil
		obj.LogsArithmeticProcessor = nil
		obj.LogsStringBuilderProcessor = nil
		obj.LogsPipelineProcessor = nil
		obj.LogsGeoIPParser = nil
		obj.LogsLookupProcessor = nil
		obj.LogsTraceRemapper = nil
		return json.Unmarshal(data, &obj.UnparsedObject)
	}
	return nil // exactly one match
}

// MarshalJSON turns data from the first non-nil pointers in the struct to JSON.
func (obj LogsProcessor) MarshalJSON() ([]byte, error) {
	if obj.LogsGrokParser != nil {
		return json.Marshal(&obj.LogsGrokParser)
	}

	if obj.LogsDateRemapper != nil {
		return json.Marshal(&obj.LogsDateRemapper)
	}

	if obj.LogsStatusRemapper != nil {
		return json.Marshal(&obj.LogsStatusRemapper)
	}

	if obj.LogsServiceRemapper != nil {
		return json.Marshal(&obj.LogsServiceRemapper)
	}

	if obj.LogsMessageRemapper != nil {
		return json.Marshal(&obj.LogsMessageRemapper)
	}

	if obj.LogsAttributeRemapper != nil {
		return json.Marshal(&obj.LogsAttributeRemapper)
	}

	if obj.LogsURLParser != nil {
		return json.Marshal(&obj.LogsURLParser)
	}

	if obj.LogsUserAgentParser != nil {
		return json.Marshal(&obj.LogsUserAgentParser)
	}

	if obj.LogsCategoryProcessor != nil {
		return json.Marshal(&obj.LogsCategoryProcessor)
	}

	if obj.LogsArithmeticProcessor != nil {
		return json.Marshal(&obj.LogsArithmeticProcessor)
	}

	if obj.LogsStringBuilderProcessor != nil {
		return json.Marshal(&obj.LogsStringBuilderProcessor)
	}

	if obj.LogsPipelineProcessor != nil {
		return json.Marshal(&obj.LogsPipelineProcessor)
	}

	if obj.LogsGeoIPParser != nil {
		return json.Marshal(&obj.LogsGeoIPParser)
	}

	if obj.LogsLookupProcessor != nil {
		return json.Marshal(&obj.LogsLookupProcessor)
	}

	if obj.LogsTraceRemapper != nil {
		return json.Marshal(&obj.LogsTraceRemapper)
	}

	if obj.UnparsedObject != nil {
		return json.Marshal(obj.UnparsedObject)
	}
	return nil, nil // no data in oneOf schemas
}

// GetActualInstance returns the actual instance.
func (obj *LogsProcessor) GetActualInstance() interface{} {
	if obj.LogsGrokParser != nil {
		return obj.LogsGrokParser
	}

	if obj.LogsDateRemapper != nil {
		return obj.LogsDateRemapper
	}

	if obj.LogsStatusRemapper != nil {
		return obj.LogsStatusRemapper
	}

	if obj.LogsServiceRemapper != nil {
		return obj.LogsServiceRemapper
	}

	if obj.LogsMessageRemapper != nil {
		return obj.LogsMessageRemapper
	}

	if obj.LogsAttributeRemapper != nil {
		return obj.LogsAttributeRemapper
	}

	if obj.LogsURLParser != nil {
		return obj.LogsURLParser
	}

	if obj.LogsUserAgentParser != nil {
		return obj.LogsUserAgentParser
	}

	if obj.LogsCategoryProcessor != nil {
		return obj.LogsCategoryProcessor
	}

	if obj.LogsArithmeticProcessor != nil {
		return obj.LogsArithmeticProcessor
	}

	if obj.LogsStringBuilderProcessor != nil {
		return obj.LogsStringBuilderProcessor
	}

	if obj.LogsPipelineProcessor != nil {
		return obj.LogsPipelineProcessor
	}

	if obj.LogsGeoIPParser != nil {
		return obj.LogsGeoIPParser
	}

	if obj.LogsLookupProcessor != nil {
		return obj.LogsLookupProcessor
	}

	if obj.LogsTraceRemapper != nil {
		return obj.LogsTraceRemapper
	}

	// all schemas are nil
	return nil
}

// NullableLogsProcessor handles when a null is used for LogsProcessor.
type NullableLogsProcessor struct {
	value *LogsProcessor
	isSet bool
}

// Get returns the associated value.
func (v NullableLogsProcessor) Get() *LogsProcessor {
	return v.value
}

// Set changes the value and indicates it's been called.
func (v *NullableLogsProcessor) Set(val *LogsProcessor) {
	v.value = val
	v.isSet = true
}

// IsSet returns whether Set has been called.
func (v NullableLogsProcessor) IsSet() bool {
	return v.isSet
}

// Unset sets the value to nil and resets the set flag/
func (v *NullableLogsProcessor) Unset() {
	v.value = nil
	v.isSet = false
}

// NewNullableLogsProcessor initializes the struct as if Set has been called.
func NewNullableLogsProcessor(val *LogsProcessor) *NullableLogsProcessor {
	return &NullableLogsProcessor{value: val, isSet: true}
}

// MarshalJSON serializes the associated value.
func (v NullableLogsProcessor) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

// UnmarshalJSON deserializes the payload and sets the flag as if Set has been called.
func (v *NullableLogsProcessor) UnmarshalJSON(src []byte) error {
	v.isSet = true

	// this object is nullable so check if the payload is null or empty string
	if string(src) == "" || string(src) == "{}" {
		return nil
	}

	return json.Unmarshal(src, &v.value)
}
