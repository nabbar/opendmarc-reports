// Copyright © 2018 Nicolas JUHEL (https://github.com/nabbar/)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"

	"github.com/nabbar/opendmarc-reports/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:     "config <file path to be generated>",
	Example: "config ~/.dmarc-reports.yml",
	Short:   "Generate config file",
	Long: `Generate a configuration file based on
giving existing config flag
override by passed flag in command line
and completed with default for non existing values
.`,
	Run: func(cmd *cobra.Command, args []string) {
		logger.DebugLevel.LogData("Viper Settings : ", viper.AllSettings())
		viper.WriteConfigAs(args[0])
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("arguments missing : requires the destination file path")
		} else if len(args) > 1 {
			return errors.New("arguments error : too many file path specify")
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
