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

	"github.com/flanksource/commons/logger"
	"github.com/spf13/cobra"
	"gopkg.in/flanksource/yaml.v3"
)

var jsonFormat bool
var pretty bool

// root represents the base command when called without any subcommands
var root = &cobra.Command{
	Use:   "yaml [file|-]",
	Short: "Processes a YAML file using the flanksource/yaml.v3 with added support for !!env and !!template tags",
}

func main() {
	root.Args = cobra.ExactArgs(1)
	root.PersistentFlags().BoolVarP(&jsonFormat, "json", "j", false, "Output JSON")
	root.PersistentFlags().BoolVarP(&pretty, "pretty", "p", true, "Pretty print ")

	root.Run = func(cmd *cobra.Command, args []string) {
		cwd, _ := os.Getwd()
		defer func() {
			_ = os.Chdir(cwd)
		}()
		file := args[0]
		if file == "-" {
			file = "/dev/stdin"
		}
		data, err := ioutil.ReadFile(file)
		if err != nil {
			logger.Fatalf("Error reading %s: %v", file, err)
		}
		if file != "/dev/stdin" {
			dir := path.Dir(file)
			if err := os.Chdir(dir); err != nil {
				logger.Fatalf("Error changing dir to %s: %v", dir, err)
			}
		}
		var o interface{}
		reader := bytes.NewReader(data)
		decoder := yaml.NewDecoder(reader)

		if err := decoder.Decode(&o); err != nil {
			logger.Fatalf("Error reading %s: %v", file, err)
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
