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

package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
	"github.com/spf13/pflag"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

var (
	kedaVersion   = "1.0"
	schemaVersion = 1.0
)

// Identifier for the creator function of the scaler
// e.g. NewRedisScaler, NewSeleniumGridScaler
const (
	creatorSymbol = "New"
)

// Metadata is a struct that represents each field of the trigger metadata
type Metadata struct {
	// Name is the name of the field
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Type is the variable type of the field
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Optional is a boolean that indicates if the field is optional
	Optional bool `json:"optional,omitempty" yaml:"optional,omitempty"`

	// Default is the default value of the field if exists
	Default string `json:"default,omitempty" yaml:"default,omitempty"`

	// AllowedValue is a list of allowed values for the field
	AllowedValue []string `json:"allowedValue,omitempty" yaml:"allowedValue,omitempty"`

	// Deprecated is a string that indicates if the field is deprecated
	Deprecated string `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`

	// DeprecatedAnnounce is a string that indicates the deprecation message
	DeprecatedAnnounce string `json:"deprecatedAnnounce,omitempty" yaml:"deprecatedAnnounce,omitempty"`

	// Separator is the symbol that separates the value of the field
	Separator string `json:"separator,omitempty" yaml:"separator,omitempty"`

	// ExclusiveSet is a list of fields that are exclusive with the field
	ExclusiveSet []string `json:"exclusiveSet,omitempty" yaml:"exclusiveSet,omitempty"`

	// RangeSeparator is the symbol that indicates the range of the field
	RangeSeparator string `json:"rangeSeparator,omitempty" yaml:"rangeSeparator,omitempty"`

	// CanReadFromEnv is a boolean that indicates if the field can be read from the environment
	CanReadFromEnv bool `json:"canReadFromEnv,omitempty" yaml:"canReadFromEnv,omitempty"`

	// CanReadFromAuth is a boolean that indicates if the field can be read from the TriggerAuthentication
	CanReadFromAuth bool `json:"canReadFromTAuthe,omitempty" yaml:"canReadFromAuth,omitempty"`
}

// TriggerMetadataSchema is a struct that represents the metadata of a trigger
type TriggerMetadataSchema struct {
	// Type is the name of the trigger
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Metadata is a list of fields of the trigger
	Metadata []Metadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// FullMetadataSchema is a complete schema of the trigger metadata
type FullMetadataSchema struct {
	// KedaVersion is the version of the current KEDA
	KedaVersion string `json:"kedaVersion,omitempty" yaml:"kedaVersion,omitempty"`

	// SchemaVersion is the version of the schema
	SchemaVersion float64 `json:"schemaVersion,omitempty" yaml:"schemaVersion,omitempty"`

	// Triggers is a list of triggers
	Triggers []TriggerMetadataSchema `json:"triggers,omitempty" yaml:"triggers,omitempty"`
}

// aggregateSchemaStruct is a function that aggregates the info from different scaler structs and generates a schema
// scalersSelectors is a map that contains the name of the scaler and the name of the scaler creator function from the scalers_builder file
// kedaScalerStructs is the structs of the scalers that are tagged with `keda`
// kedaReferenceKedaTagStructs is the sub structs that are referenced by the keda tagged structs
func aggregateSchemaStruct(scalerSelectors map[string]string, kedaScalerStructs map[string]*ast.StructType, otherReferenceKedaTagStructs map[string]*ast.StructType, outputFilePath string) (err error) {

	triggerMetadataSchemas := []TriggerMetadataSchema{}

	for creatorName, scalerStructs := range kedaScalerStructs {
		metadataFields, err := generateMetadataFields(scalerStructs, otherReferenceKedaTagStructs)
		if err != nil {
			fmt.Printf("Error generating metadata fields with creator %s: %s\n", creatorName, err)
			continue
		}

		// Find which trigger names the creator is called by and construct the metadata schema
		for triggerName, selectorName := range scalerSelectors {
			if selectorName == creatorName {
				triggerMetadataSchema := TriggerMetadataSchema{}
				triggerMetadataSchema.Type = triggerName
				triggerMetadataSchema.Metadata = metadataFields
				triggerMetadataSchemas = append(triggerMetadataSchemas, triggerMetadataSchema)
				fmt.Printf("Scaler Metadata Schema Added: %s\n", triggerName)
			}

		}
	}

	// Combine all the metadata schemas into a complete schema
	fullMetadataSchema := FullMetadataSchema{
		KedaVersion:   kedaVersion,
		SchemaVersion: schemaVersion,
		Triggers:      triggerMetadataSchemas,
	}

	yamlData, err := yaml.Marshal(fullMetadataSchema)
	if err != nil {
		return err
	}

	fileName := outputFilePath + "scaler-metadata-schemas.yaml"
	err = os.WriteFile(fileName, yamlData, 0644)
	if err != nil {
		return err
	}
	return nil
}

// generateMetadataFields is a function that generates the metadata fields of a scaler struct
func generateMetadataFields(structType *ast.StructType, otherReferenceKedaTagStructs map[string]*ast.StructType) ([]Metadata, error) {

	triggerMetadata := []Metadata{}

	// get the tag of each field and generate the metadata
	for _, commentGroup := range structType.Fields.List {
		if commentGroup.Tag == nil || commentGroup.Tag.Value == "" {
			continue
		}
		metadataList, hasSubstruct, err := generateMetadatasFromTag(commentGroup.Tag.Value)

		if err != nil {
			fmt.Printf("Error generating metadata fields from tag value: %s, err: %s\n", commentGroup.Tag.Value, err)
			continue
		}

		if !hasSubstruct {
			triggerMetadata = append(triggerMetadata, metadataList...)
			continue
		}

		// If the field has a substruct, try to find substruct from reference structs
		s, ok := commentGroup.Type.(*ast.Ident)
		if !ok {
			continue
		}
		if otherReferenceKedaTagStructs[s.Name] != nil {
			subStructMetadataField, err := generateMetadataFields(otherReferenceKedaTagStructs[s.Name], otherReferenceKedaTagStructs)
			if err == nil {
				triggerMetadata = append(triggerMetadata, subStructMetadataField...)
			}
		}
	}

	return triggerMetadata, nil
}

// generateMetadatasFromTag is a function that generates the metadata field from tag
func generateMetadatasFromTag(tag string) ([]Metadata, bool, error) {
	var fieldNames []string
	metadata := Metadata{Type: "string"}
	tagSplit := strings.Split(strings.Trim(strings.Split(strings.Trim(tag, "`"), ":")[1], "\""), scalersconfig.TagSeparator)

	if len(tagSplit) == 1 && tagSplit[0] == scalersconfig.OptionalTag {
		return nil, true, nil
	}

	for _, ts := range tagSplit {
		tsplit := strings.Split(ts, scalersconfig.TagKeySeparator)
		tsplit[0] = strings.TrimSpace(tsplit[0])
		switch tsplit[0] {
		case scalersconfig.OptionalTag:
			if len(tsplit) == 1 {
				metadata.Optional = true
			}
			if len(tsplit) > 1 {
				optional, err := strconv.ParseBool(strings.TrimSpace(tsplit[1]))
				if err != nil {
					return nil, false, fmt.Errorf("error parsing optional value %s: %w", tsplit[1], err)
				}
				metadata.Optional = optional
			}
		case scalersconfig.OrderTag:
			if len(tsplit) > 1 {
				order := strings.Split(tsplit[1], scalersconfig.TagValueSeparator)
				for _, po := range order {
					poTyped := scalersconfig.ParsingOrder(strings.TrimSpace(po))
					if !scalersconfig.AllowedParsingOrderMap[poTyped] {
						apo := maps.Keys(scalersconfig.AllowedParsingOrderMap)
						slices.Sort(apo)
						return nil, false, fmt.Errorf("unknown parsing order value %s, has to be one of %s", po, apo)
					}
					if poTyped == scalersconfig.ResolvedEnv {
						metadata.CanReadFromEnv = true
					} else if poTyped == scalersconfig.AuthParams {
						metadata.CanReadFromAuth = true
					}
				}
			}
		case scalersconfig.NameTag:
			if len(tsplit) > 1 {
				fieldNames = strings.Split(strings.TrimSpace(tsplit[1]), scalersconfig.TagValueSeparator)
			}
		case scalersconfig.DeprecatedTag:
			if len(tsplit) == 1 {
				metadata.Deprecated = scalersconfig.DeprecatedTag
			} else {
				metadata.Deprecated = strings.TrimSpace(tsplit[1])
			}
		case scalersconfig.DeprecatedAnnounceTag:
			if len(tsplit) == 1 {
				metadata.DeprecatedAnnounce = scalersconfig.DeprecatedAnnounceTag
			} else {
				metadata.DeprecatedAnnounce = strings.TrimSpace(tsplit[1])
			}
		case scalersconfig.DefaultTag:
			if len(tsplit) > 1 {
				metadata.Default = strings.TrimSpace(tsplit[1])
			}
		case scalersconfig.EnumTag:
			if len(tsplit) > 1 {
				metadata.AllowedValue = strings.Split(tsplit[1], scalersconfig.TagValueSeparator)
			}
		case scalersconfig.ExclusiveSetTag:
			if len(tsplit) > 1 {
				metadata.ExclusiveSet = strings.Split(tsplit[1], scalersconfig.TagValueSeparator)
			}
		case scalersconfig.RangeTag:
			if len(tsplit) == 1 {
				metadata.RangeSeparator = "-"
			}
			if len(tsplit) == 2 {
				metadata.RangeSeparator = strings.TrimSpace(tsplit[1])
			}
		case scalersconfig.SeparatorTag:
			if len(tsplit) > 1 {
				metadata.Separator = strings.TrimSpace(tsplit[1])
			}
		case "":
			continue
		default:
			return nil, false, fmt.Errorf("unknown tag param %s: %s", tsplit[0], tag)
		}
	}

	if len(fieldNames) == 0 {
		return nil, false, fmt.Errorf("Fieldname doesn't exist in tag value")
	}

	metadatas := []Metadata{}
	for _, fieldName := range fieldNames {
		metadata.Name = fieldName
		metadatas = append(metadatas, metadata)
	}

	return metadatas, false, nil
}

// getBuildScalerCalls is a function that gets the map of trigger names and the creator function names from the scalers_builder file
func getBuildScalerCalls(fileName string) (map[string]string, error) {
	scalerCallers := map[string]string{}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", string(data), 0)
	if err != nil {
		return nil, err
	}

	// Try to find the "triggerType" switch statement and get the trigger names and creator function names
	ast.Inspect(f, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.SwitchStmt:
			s, ok := t.Tag.(*ast.Ident)
			if !ok || s.Name != "triggerType" {
				break
			}
			// retrieve the creator function name from each case statment
			for _, casesAst := range t.Body.List {
				c, ok := casesAst.(*ast.CaseClause)

				if !ok || len(c.List) == 0 || len(c.Body) == 0 {
					continue
				}

				basic, ok := c.List[0].(*ast.BasicLit)
				if !ok {
					continue
				}

				r, ok := c.Body[0].(*ast.ReturnStmt)
				if !ok {
					continue
				}

				if len(r.Results) == 0 {
					continue
				}
				caller, ok := r.Results[0].(*ast.CallExpr)
				if !ok {
					continue
				}

				expr, ok := caller.Fun.(*ast.SelectorExpr)
				if !ok {
					continue
				}

				scalerCallers[strings.Trim(basic.Value, "\"")] = expr.Sel.Name
			}

		}
		return true
	})
	return scalerCallers, nil
}

// getAllKedaTagedStructs is a function that gets all the structs that are tagged with `keda` from the scalers files
func getAllKedaTagedStructs(dir string) (map[string]*ast.StructType, map[string]*ast.StructType) {
	// loop all files in the scalers directory and get the structs that are tagged with `keda`
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	kedaScalerStructs := map[string]*ast.StructType{}
	kedaTagStructs := map[string]*ast.StructType{}

	for _, e := range entries {
		if e.IsDir() {
			getAllKedaTagedStructs(dir + "/" + e.Name())
			continue
		}

		data, err := os.ReadFile(dir + "/" + e.Name())
		if err != nil {
			continue
		}

		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", string(data), 0)
		if err != nil {
			continue
		}

		var scalerStructs *ast.StructType

		for _, decl := range f.Decls {

			switch v := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range v.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}

					structType, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}

					hasTriggerIndex := false
					for _, v := range structType.Fields.List {
						// Find `triggerIndex` field and identify the struct as a scaler struct, otherwise identify the struct as an abstract substruct
						if len(v.Names) > 0 && v.Names[0].Name == "triggerIndex" {
							hasTriggerIndex = true
							continue
						}

						if v.Tag == nil || v.Tag.Value == "" {
							continue
						}

						tagvalue := strings.Split(strings.Trim(v.Tag.Value, "`"), ":")

						// if the tag is `keda`, add the struct to the kedaTagStructs map
						if len(tagvalue) > 0 && tagvalue[0] == "keda" {
							kedaTagStructs[ts.Name.Name] = structType
						}
					}

					if hasTriggerIndex {
						scalerStructs = kedaTagStructs[ts.Name.Name]
						delete(kedaTagStructs, ts.Name.Name)
					}
				}
			case *ast.FuncDecl:
				if strings.HasPrefix(v.Name.Name, creatorSymbol) && scalerStructs != nil {
					kedaScalerStructs[v.Name.Name] = scalerStructs
				}
			}
		}
	}

	return kedaScalerStructs, kedaTagStructs
}

func main() {
	var builderFilePath string
	var scalersFilesDirPath string
	var specifyScaler string
	var outputFilePath string
	pflag.StringVar(&kedaVersion, "keda-version", "1.0", "Set the version of current KEDA in schema.")
	pflag.StringVar(&builderFilePath, "scalers-builder-file", "../pkg/scaling/scalers_builder.go", "The file that exists `buildScaler` func.")
	pflag.StringVar(&scalersFilesDirPath, "scalers-files-dir", "../pkg/scalers", "The directory that exists all scalers' files.")
	pflag.StringVar(&specifyScaler, "specify-scaler", "", "Specify scaler name.")
	pflag.StringVar(&outputFilePath, "output-file-path", "./", "triggerMetadata.yaml output file path.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	scalerSelectors, err := getBuildScalerCalls(builderFilePath)
	if err != nil {
		fmt.Print("error")
	}

	if specifyScaler != "" {
		if scalerSelectors[specifyScaler] != "" {
			scalerSelectors = map[string]string{specifyScaler: scalerSelectors[specifyScaler]}
		} else {
			fmt.Println("Cannot find the specified scaler")
			os.Exit(0)
		}
	}

	kedaTagStructs, otherReferenceStructs := getAllKedaTagedStructs(scalersFilesDirPath)
	err = aggregateSchemaStruct(scalerSelectors, kedaTagStructs, otherReferenceStructs, outputFilePath)
	if err != nil {
		fmt.Print("error")
	}
}
