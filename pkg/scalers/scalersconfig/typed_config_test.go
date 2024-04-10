/*
Copyright 2024 The KEDA Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package scalersconfig

import (
	"net/url"
	"testing"

	. "github.com/onsi/gomega"
)

// TestBasicTypedConfig tests the basic types for typed config
func TestBasicTypedConfig(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"stringVal":       "value1",
			"intVal":          "1",
			"boolValFromEnv":  "boolVal",
			"floatValFromEnv": "floatVal",
		},
		ResolvedEnv: map[string]string{
			"boolVal":  "true",
			"floatVal": "1.1",
		},
		AuthParams: map[string]string{
			"auth": "authValue",
		},
	}

	type testStruct struct {
		StringVal string  `keda:"name=stringVal, parsingOrder=triggerMetadata"`
		IntVal    int     `keda:"name=intVal,    parsingOrder=triggerMetadata"`
		BoolVal   bool    `keda:"name=boolVal,   parsingOrder=resolvedEnv"`
		FloatVal  float64 `keda:"name=floatVal,  parsingOrder=resolvedEnv"`
		AuthVal   string  `keda:"name=auth,      parsingOrder=authParams"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())

	Expect(ts.StringVal).To(Equal("value1"))
	Expect(ts.IntVal).To(Equal(1))
	Expect(ts.BoolVal).To(BeTrue())
	Expect(ts.FloatVal).To(Equal(1.1))
	Expect(ts.AuthVal).To(Equal("authValue"))
}

// TestParsingOrder tests the parsing order
func TestParsingOrder(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"stringVal":       "value1",
			"intVal":          "1",
			"intValFromEnv":   "intVal",
			"floatVal":        "1.1",
			"floatValFromEnv": "floatVal",
		},
		ResolvedEnv: map[string]string{
			"stringVal": "value2",
			"intVal":    "2",
			"floatVal":  "2.2",
		},
	}

	type testStruct struct {
		StringVal string  `keda:"name=stringVal, parsingOrder=resolvedEnv;triggerMetadata"`
		IntVal    int     `keda:"name=intVal,    parsingOrder=triggerMetadata;resolvedEnv"`
		FloatVal  float64 `keda:"name=floatVal,  parsingOrder=resolvedEnv;triggerMetadata"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())

	Expect(ts.StringVal).To(Equal("value1"))
	Expect(ts.IntVal).To(Equal(1))
	Expect(ts.FloatVal).To(Equal(2.2))
}

// TestOptional tests the optional tag
func TestOptional(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"stringVal": "value1",
		},
	}

	type testStruct struct {
		StringVal          string `keda:"name=stringVal, parsingOrder=triggerMetadata"`
		IntValOptional     int    `keda:"name=intVal,    parsingOrder=triggerMetadata, optional"`
		IntValAlsoOptional int    `keda:"name=intVal,    parsingOrder=triggerMetadata, optional=true"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())

	Expect(ts.StringVal).To(Equal("value1"))
	Expect(ts.IntValOptional).To(Equal(0))
	Expect(ts.IntValAlsoOptional).To(Equal(0))
}

// TestMissing tests the missing parameter for compulsory tag
func TestMissing(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{}

	type testStruct struct {
		StringVal string `keda:"name=stringVal, parsingOrder=triggerMetadata"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(MatchError(`missing required parameter "stringVal" in [triggerMetadata]`))
}

// TestDeprecated tests the deprecated tag
func TestDeprecated(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"stringVal": "value1",
		},
	}

	type testStruct struct {
		StringVal string `keda:"name=stringVal, parsingOrder=triggerMetadata, deprecated=deprecated"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(MatchError(`parameter "stringVal" is deprecated`))

	sc2 := &ScalerConfig{
		TriggerMetadata: map[string]string{},
	}

	ts2 := testStruct{}
	err = sc2.TypedConfig(&ts2)
	Expect(err).To(BeNil())
}

// TestDefaultValue tests the default tag
func TestDefaultValue(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"stringVal": "value1",
		},
	}

	type testStruct struct {
		BoolVal    bool   `keda:"name=boolVal,    parsingOrder=triggerMetadata, optional, default=true"`
		StringVal  string `keda:"name=stringVal,  parsingOrder=triggerMetadata, optional, default=d"`
		StringVal2 string `keda:"name=stringVal2, parsingOrder=triggerMetadata, optional, default=d"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())

	Expect(ts.BoolVal).To(Equal(true))
	Expect(ts.StringVal).To(Equal("value1"))
	Expect(ts.StringVal2).To(Equal("d"))
}

// TestMap tests the map type
func TestMap(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"mapVal": "key1=1,key2=2",
		},
	}

	type testStruct struct {
		MapVal map[string]int `keda:"name=mapVal, parsingOrder=triggerMetadata"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())
	Expect(ts.MapVal).To(HaveLen(2))
	Expect(ts.MapVal["key1"]).To(Equal(1))
	Expect(ts.MapVal["key2"]).To(Equal(2))
}

// TestSlice tests the slice type
func TestSlice(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"sliceVal":           "1,2,3",
			"sliceValWithSpaces": "1, 2, 3",
		},
	}

	type testStruct struct {
		SliceVal           []int `keda:"name=sliceVal, parsingOrder=triggerMetadata"`
		SliceValWithSpaces []int `keda:"name=sliceValWithSpaces, parsingOrder=triggerMetadata"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())
	Expect(ts.SliceVal).To(HaveLen(3))
	Expect(ts.SliceVal[0]).To(Equal(1))
	Expect(ts.SliceVal[1]).To(Equal(2))
	Expect(ts.SliceVal[2]).To(Equal(3))
	Expect(ts.SliceValWithSpaces).To(HaveLen(3))
	Expect(ts.SliceValWithSpaces[0]).To(Equal(1))
	Expect(ts.SliceValWithSpaces[1]).To(Equal(2))
	Expect(ts.SliceValWithSpaces[2]).To(Equal(3))
}

// TestEnum tests the enum type
func TestEnum(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"enumVal":   "value1",
			"enumSlice": "value1, value2",
		},
	}

	type testStruct struct {
		EnumVal   string   `keda:"name=enumVal,   parsingOrder=triggerMetadata, enum=value1;value2"`
		EnumSlice []string `keda:"name=enumSlice, parsingOrder=triggerMetadata, enum=value1;value2, optional"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())
	Expect(ts.EnumVal).To(Equal("value1"))
	Expect(ts.EnumSlice).To(HaveLen(2))
	Expect(ts.EnumSlice).To(ConsistOf("value1", "value2"))

	sc2 := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"enumVal": "value3",
		},
	}

	ts2 := testStruct{}
	err = sc2.TypedConfig(&ts2)
	Expect(err).To(MatchError(`parameter "enumVal" value "value3" must be one of [value1 value2]`))
}

// TestExclusive tests the exclusive type
func TestExclusive(t *testing.T) {
	RegisterTestingT(t)

	type testStruct struct {
		IntVal []int `keda:"name=intVal,    parsingOrder=triggerMetadata, exclusive=1;4;5"`
	}

	sc := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"intVal": "1,2,3",
		},
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())

	sc2 := &ScalerConfig{
		TriggerMetadata: map[string]string{
			"intVal": "1,4",
		},
	}

	ts2 := testStruct{}
	err = sc2.TypedConfig(&ts2)
	Expect(err).To(MatchError(`parameter "intVal" value "1,4" must contain only one of [1 4 5]`))
}

// TestURLValues tests the url.Values type
func TestURLValues(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		AuthParams: map[string]string{
			"endpointParams": "key1=value1&key2=value2&key1=value3",
		},
	}

	type testStruct struct {
		EndpointParams url.Values `keda:"name=endpointParams, parsingOrder=authParams"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())
	Expect(ts.EndpointParams).To(HaveLen(2))
	Expect(ts.EndpointParams).To(HaveKey("key1"))
	Expect(ts.EndpointParams).To(HaveKey("key2"))
	Expect(ts.EndpointParams["key1"]).To(ConsistOf("value1", "value3"))
	Expect(ts.EndpointParams["key2"]).To(ConsistOf("value2"))
}

// TestGenericMap tests the generic map type that is structurally similar to url.Values
func TestGenericMap(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		AuthParams: map[string]string{
			"endpointParams": "key1=value1,key2=value2,key3=value3",
		},
	}

	// structurally similar to url.Values but should behave as generic map
	type testStruct struct {
		EndpointParams map[string][]string `keda:"name=endpointParams, parsingOrder=authParams"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())
	Expect(ts.EndpointParams).To(HaveLen(3))
	Expect(ts.EndpointParams).To(HaveKey("key1"))
	Expect(ts.EndpointParams).To(HaveKey("key2"))
	Expect(ts.EndpointParams).To(HaveKey("key3"))
	Expect(ts.EndpointParams["key1"]).To(ConsistOf("value1"))
	Expect(ts.EndpointParams["key2"]).To(ConsistOf("value2"))
	Expect(ts.EndpointParams["key3"]).To(ConsistOf("value3"))
}

// TestNestedStruct tests the nested struct type
func TestNestedStruct(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		AuthParams: map[string]string{
			"username": "user",
			"password": "pass",
		},
	}

	type basicAuth struct {
		Username string `keda:"name=username, parsingOrder=authParams"`
		Password string `keda:"name=password, parsingOrder=authParams"`
	}

	type testStruct struct {
		BA basicAuth `keda:""`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())
	Expect(ts.BA.Username).To(Equal("user"))
	Expect(ts.BA.Password).To(Equal("pass"))
}

// TestEmbeddedStruct tests the embedded struct type
func TestEmbeddedStruct(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		AuthParams: map[string]string{
			"username": "user",
			"password": "pass",
		},
	}

	type testStruct struct {
		BasicAuth struct {
			Username string `keda:"name=username, parsingOrder=authParams"`
			Password string `keda:"name=password, parsingOrder=authParams"`
		} `keda:""`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())
	Expect(ts.BasicAuth.Username).To(Equal("user"))
	Expect(ts.BasicAuth.Password).To(Equal("pass"))
}

// TestWrongNestedStruct tests the wrong nested type
func TestWrongNestedStruct(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		AuthParams: map[string]string{
			"username": "user",
			"password": "pass",
		},
	}

	type testStruct struct {
		WrongNesting int `keda:""`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(MatchError(`nested parameter "WrongNesting" must be a struct, has kind "int"`))
}

// TestNestedOptional tests the nested optional type
func TestNestedOptional(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		AuthParams: map[string]string{
			"username": "user",
		},
	}

	type basicAuth struct {
		Username                   string `keda:"name=username, parsingOrder=authParams"`
		Password                   string `keda:"name=password, parsingOrder=authParams, optional"`
		AlsoOptionalThanksToParent string `keda:"name=optional, parsingOrder=authParams"`
	}

	type testStruct struct {
		BA basicAuth `keda:"optional"`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())
	Expect(ts.BA.Username).To(Equal("user"))
	Expect(ts.BA.Password).To(Equal(""))
	Expect(ts.BA.AlsoOptionalThanksToParent).To(Equal(""))
}

// TestNestedPointer tests the nested pointer type
func TestNestedPointer(t *testing.T) {
	RegisterTestingT(t)

	sc := &ScalerConfig{
		AuthParams: map[string]string{
			"username": "user",
			"password": "pass",
		},
	}

	type basicAuth struct {
		Username string `keda:"name=username, parsingOrder=authParams"`
		Password string `keda:"name=password, parsingOrder=authParams"`
	}

	type testStruct struct {
		BA *basicAuth `keda:""`
	}

	ts := testStruct{}
	err := sc.TypedConfig(&ts)
	Expect(err).To(BeNil())
	Expect(ts.BA).ToNot(BeNil())
	Expect(ts.BA.Username).To(Equal("user"))
	Expect(ts.BA.Password).To(Equal("pass"))
}
