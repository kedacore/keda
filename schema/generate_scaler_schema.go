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
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"

	"github.com/kedacore/keda/v2/pkg/scalers/scalersconfig"
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

// Parameters is a struct that represents each field of the scaler parameters
type Parameters struct {
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

	// MetadataVariableReadable is a boolean that indicates if the field can be read from the environment
	MetadataVariableReadable bool `json:"metadataVariableReadable,omitempty" yaml:"metadataVariableReadable,omitempty"`

	// EnvVariableReadable is a boolean that indicates if the field can be read from the environment
	EnvVariableReadable bool `json:"envVariableReadable,omitempty" yaml:"envVariableReadable,omitempty"`

	// TriggerAuthenticationVariableReadable is a boolean that indicates if the field can be read from the TriggerAuthentication
	TriggerAuthenticationVariableReadable bool `json:"triggerAuthenticationVariableReadable,omitempty" yaml:"triggerAuthenticationVariableReadable,omitempty"`
}

// ScalerMetadataSchema is a struct that represents the metadata of a scler
type ScalerMetadataSchema struct {
	// Type is the name of the scaler
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// Parameters is a list of fields of the scaler
	Parameters []Parameters `json:"parameters,omitempty" yaml:"parameters,omitempty"`
}

// FullMetadataSchema is a complete schema of the scaler metadata
type FullMetadataSchema struct {
	// KedaVersion is the version of the current KEDA
	KedaVersion string `json:"kedaVersion,omitempty" yaml:"kedaVersion,omitempty"`

	// SchemaVersion is the version of the schema
	SchemaVersion float64 `json:"schemaVersion,omitempty" yaml:"schemaVersion,omitempty"`

	// Scalers is a list of scalers
	Scalers []ScalerMetadataSchema `json:"scalers,omitempty" yaml:"scalers,omitempty"`
}

// aggregateSchemaStruct is a function that aggregates the info from different scaler structs and generates a schema
// scalersSelectors is a map that contains the name of the scaler and the name of the scaler creator function from the scalers_builder file
// kedaScalerStructs is the structs of the scalers that are tagged with `keda`
// kedaReferenceKedaTagStructs is the sub structs that are referenced by the keda tagged structs
func aggregateSchemaStruct(scalerSelectors map[string]string, kedaScalerStructs map[string]*ast.StructType, otherReferenceKedaTagStructs map[string]*ast.StructType, outputFileName string, outputFilePath string, outputFileFormat string) (err error) {
	scalerMetadataSchemas := []ScalerMetadataSchema{}
	sortedScalerCreatorNames := []string{}
	for k := range kedaScalerStructs {
		sortedScalerCreatorNames = append(sortedScalerCreatorNames, k)
	}
	sort.Strings(sortedScalerCreatorNames)

	sortedScalerNames := []string{}
	for k := range scalerSelectors {
		sortedScalerNames = append(sortedScalerNames, k)
	}
	sort.Strings(sortedScalerNames)

	for _, creatorName := range sortedScalerCreatorNames {
		metadataFields := generateMetadataFields(kedaScalerStructs[creatorName], otherReferenceKedaTagStructs)
		if len(metadataFields) == 0 {
			fmt.Printf("Error generating metadata fields with creator %s: %s\n", creatorName, err)
			continue
		}

		// Find which scaler names the creator is called by and construct the metadata schema
		for _, scalerName := range sortedScalerNames {
			if scalerSelectors[scalerName] == creatorName {
				scalerMetadataSchema := ScalerMetadataSchema{}
				scalerMetadataSchema.Type = scalerName
				scalerMetadataSchema.Parameters = metadataFields
				scalerMetadataSchemas = append(scalerMetadataSchemas, scalerMetadataSchema)
				fmt.Printf("Scaler Metadata Schema Added: %s\n", scalerName)
			}
		}
	}

	// Combine all the metadata schemas into a complete schema
	fullMetadataSchema := FullMetadataSchema{
		KedaVersion:   kedaVersion,
		SchemaVersion: schemaVersion,
		Scalers:       scalerMetadataSchemas,
	}

	if _, err := os.Stat(outputFilePath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(outputFilePath, os.ModePerm)
		if err != nil {
			return err
		}
	}

	switch outputFileFormat {
	case "yaml":
		filedata, err := yaml.Marshal(fullMetadataSchema)
		if err != nil {
			return err
		}

		fileName := outputFilePath + "/" + outputFileName + ".yaml"
		err = os.WriteFile(fileName, filedata, 0644)
		if err != nil {
			return err
		}
	case "json":
		filedata, err := json.MarshalIndent(fullMetadataSchema, "", "    ")
		if err != nil {
			return err
		}

		filedata = append(filedata, '\n')
		fileName := outputFilePath + "/" + outputFileName + ".json"
		err = os.WriteFile(fileName, filedata, 0644)
		if err != nil {
			return err
		}
	case "both":

		filedata, err := yaml.Marshal(fullMetadataSchema)
		if err != nil {
			return err
		}

		fileName := outputFilePath + "/" + outputFileName + ".yaml"
		err = os.WriteFile(fileName, filedata, 0644)
		if err != nil {
			return err
		}

		filedata, err = json.MarshalIndent(fullMetadataSchema, "", "    ")
		if err != nil {
			return err
		}

		filedata = append(filedata, '\n')
		fileName = outputFilePath + "/" + outputFileName + ".json"
		err = os.WriteFile(fileName, filedata, 0644)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("output file format %s is not supported", outputFileFormat)
	}

	return err
}

// generateMetadataFields is a function that generates the metadata fields of a scaler struct
func generateMetadataFields(structType *ast.StructType, otherReferenceKedaTagStructs map[string]*ast.StructType) []Parameters {
	scalerMetadata := []Parameters{}

	// get the tag of each field and generate the metadata
	for _, commentGroup := range structType.Fields.List {
		if commentGroup.Tag == nil || commentGroup.Tag.Value == "" {
			continue
		}
		metadataList, hasSubstruct, err := generateMetadatas(commentGroup.Tag.Value)

		if err != nil {
			fmt.Printf("Error generating metadata fields from tag value: %s, err: %s\n", commentGroup.Tag.Value, err)
			continue
		}

		if !hasSubstruct {
			scalerMetadata = append(scalerMetadata, metadataList...)
			continue
		}

		// If the field has a substruct, try to find substruct from reference structs
		s, ok := commentGroup.Type.(*ast.Ident)
		if !ok {
			continue
		}
		if otherReferenceKedaTagStructs[s.Name] != nil {
			subStructMetadataField := generateMetadataFields(otherReferenceKedaTagStructs[s.Name], otherReferenceKedaTagStructs)
			if len(subStructMetadataField) > 0 {
				scalerMetadata = append(scalerMetadata, subStructMetadataField...)
			}
		}
	}

	return scalerMetadata
}

// generateMetadatas is a function that generates the metadata field from tag
func generateMetadatas(tag string) ([]Parameters, bool, error) {
	var fieldNames []string
	metadata := Parameters{Type: "string"}
	tagSplit := strings.Split(strings.Trim(strings.Join(strings.Split(strings.Trim(tag, "`"), ":")[1:], ":"), "\""), scalersconfig.TagSeparator)

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
				canReadFromMetadata, canReadFromEnv, canReadFromAuth, err := retrieveDataFromOrder(order)
				if err != nil {
					return nil, false, err
				}
				metadata.MetadataVariableReadable = canReadFromMetadata
				metadata.EnvVariableReadable = canReadFromEnv
				metadata.TriggerAuthenticationVariableReadable = canReadFromAuth
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
		return nil, false, fmt.Errorf("fieldname doesn't exist in tag value")
	}

	metadatas := createMetadatas(metadata, fieldNames)
	return metadatas, false, nil
}

// retrieveDataFromOrder is a function that retrieves the data from the order tag
func retrieveDataFromOrder(orders []string) (bool, bool, bool, error) {
	var canReadFromMetadata, canReadFromEnv, canReadFromAuth = false, false, false
	for _, po := range orders {
		poTyped := scalersconfig.ParsingOrder(strings.TrimSpace(po))
		if !scalersconfig.AllowedParsingOrderMap[poTyped] {
			apo := maps.Keys(scalersconfig.AllowedParsingOrderMap)
			slices.Sort(apo)
			return false, false, false, fmt.Errorf("unknown parsing order value %s, has to be one of %s", po, apo)
		}
		switch poTyped {
		case scalersconfig.TriggerMetadata:
			canReadFromMetadata = true
		case scalersconfig.ResolvedEnv:
			canReadFromEnv = true
		case scalersconfig.AuthParams:
			canReadFromAuth = true
		}
	}
	return canReadFromMetadata, canReadFromEnv, canReadFromAuth, nil
}

// createMetadatas is a function that creates the metadata with the field names
func createMetadatas(metadata Parameters, fieldNames []string) []Parameters {
	metadatas := []Parameters{}
	for _, fieldName := range fieldNames {
		metadata.Name = fieldName
		metadatas = append(metadatas, metadata)
	}
	return metadatas
}

// getBuildScalerCalls is a function that gets the map of scaler names and the creator function names from the scalers_builder file
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
			// retrieve the creator function name from each case statement
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
						if len(v.Names) > 0 && strings.EqualFold(v.Names[0].Name, "triggerIndex") {
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
	var outputFileName string
	var outputFilePath string
	var outputFormat string
	pflag.StringVar(&kedaVersion, "keda-version", "main", "Set the version of current KEDA in schema.")
	pflag.StringVar(&builderFilePath, "scalers-builder-file", "../pkg/scaling/scalers_builder.go", "The file that exists `buildScaler` func.")
	pflag.StringVar(&scalersFilesDirPath, "scalers-files-dir", "../pkg/scalers", "The directory that exists all scalers' files.")
	pflag.StringVar(&specifyScaler, "specify-scaler", "", "Specify scaler name.")
	pflag.StringVar(&outputFileName, "output-file-name", "scalers-schema", "Output file name.")
	pflag.StringVar(&outputFilePath, "output-file-path", "./", "Output file path.")
	pflag.StringVar(&outputFormat, "output-file-format", "both", "Output file format. support json and yaml.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	scalerSelectors, err := getBuildScalerCalls(builderFilePath)
	if err != nil {
		fmt.Printf("get build scaler calls error: %s\n", err)
		os.Exit(1)
	}

	if specifyScaler != "" {
		if scalerSelectors[specifyScaler] != "" {
			scalerSelectors = map[string]string{specifyScaler: scalerSelectors[specifyScaler]}
		} else {
			fmt.Println("Cannot find the specified scaler")
			os.Exit(1)
		}
	}

	kedaTagStructs, otherReferenceStructs := getAllKedaTagedStructs(scalersFilesDirPath)
	err = aggregateSchemaStruct(scalerSelectors, kedaTagStructs, otherReferenceStructs, outputFileName, outputFilePath, outputFormat)
	if err != nil {
		fmt.Printf("Error aggregating schema struct: %s\n", err)
		os.Exit(1)
	}
}
