/*
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
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/flanksource/commons/logger"
	"github.com/spf13/cobra"
	"github.com/xeipuuv/gojsonschema"
	"gopkg.in/flanksource/yaml.v3"
)

var jsonFormat bool
var pretty bool

// root represents the base command when called without any subcommands
var root = &cobra.Command{
	Use:   "yaml [file|-]",
	Short: "Processes a YAML file using the flanksource/yaml.v3 with added support for !!env and !!template tags",
}

var merge = &cobra.Command{
	Use: "merge",
	Run: func(cmd *cobra.Command, args []string) {

	},
}
var sharedFile string
var jsonSchema string
var jsonSchemaLoader gojsonschema.JSONLoader

func read(file string) (interface{}, error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		logger.Fatalf("Error reading %s: %v", file, err)
	}
	if sharedFile != "" {
		shared, err := ioutil.ReadFile(sharedFile)
		if err != nil {
			logger.Fatalf("Error reading %s: %v", sharedFile, err)
		}
		shared = append(shared, []byte("\n")...)
		data = append(shared, data...)
	}
	if file != "/dev/stdin" {
		cwd, _ := os.Getwd()
		defer func() {
			_ = os.Chdir(cwd)
		}()
		dir := path.Dir(file)
		if err := os.Chdir(dir); err != nil {
			return nil, err
		}
	}
	var o interface{}
	reader := bytes.NewReader(data)
	decoder := yaml.NewDecoder(reader)

	if err := decoder.Decode(&o); err != nil {
		return nil, err
	}
	if result, err := validate(o); err != nil {
		logger.Fatalf(err.Error())
	} else if result != "" {
		return nil, fmt.Errorf(result)
	}
	return o, nil
}

func validate(o interface{}) (string, error) {
	if jsonSchemaLoader == nil {
		return "", nil
	}

	out, err := json.Marshal(o)
	if err != nil {
		return "", fmt.Errorf("failed to marshal: %s", err)
	}
	documentLoader := gojsonschema.NewBytesLoader(out)

	result, err := gojsonschema.Validate(jsonSchemaLoader, documentLoader)
	if err != nil {
		return "", fmt.Errorf("failed to perform validation: %s", err)
	}

	if result.Valid() {
		return "", nil
	}

	s := ""
	for _, desc := range result.Errors() {
		s += fmt.Sprintf("- %s\n", desc)
	}
	return s, nil
}

func main() {
	root.Args = cobra.ExactArgs(1)

	root.PersistentFlags().BoolVarP(&jsonFormat, "json", "j", false, "Output JSON")
	root.PersistentFlags().BoolVarP(&pretty, "pretty", "p", true, "Pretty print ")
	root.PersistentFlags().StringVar(&sharedFile, "shared-file", "", "A path to a shared file that will be prepended to each yaml file before parsing ")
	root.PersistentFlags().StringVar(&jsonSchema, "json-schema", "", "A path to a JSON schema to validate each file using")
	root.Run = func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()

		if jsonSchema != "" {
			data, err := ioutil.ReadFile(jsonSchema)
			if err != nil {
				logger.Fatalf("Cannot load schema form %s: %v", jsonSchema, err)
			}
			jsonSchemaLoader = gojsonschema.NewBytesLoader(data)
		}
		var o interface{}
		var err error
		file := args[0]
		if file == "-" {
			file = "/dev/stdin"
		} else if strings.Contains(file, "*") {
			files, err := doublestar.Glob(os.DirFS(cwd), file)
			if err != nil {
				logger.Fatalf("Failed to find file matching %s: %v", file, err)
			}
			var arr []interface{}
			for _, file := range files {
				o, err = read(file)
				if err != nil {
					logger.Errorf("Failed to parse %s: %v", file, err)
				} else {
					arr = append(arr, o)
				}
			}
			o = arr
		} else {
			o, err = read(file)
		}
		if err != nil {
			logger.Fatalf("failed to parse %s: %v", file, err)
		}
		var out []byte
		if jsonFormat && pretty {
			out, err = json.MarshalIndent(o, "", "\t")
		} else if jsonFormat {
			out, err = json.Marshal(o)
		} else {
			out, err = yaml.Marshal(o)
		}
		if err != nil {
			logger.Fatalf("Error marshalling %s: %v", file, err)
		}

		fmt.Println(string(out))
	}
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
