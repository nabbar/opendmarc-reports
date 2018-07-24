package cmd

import (
	"fmt"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/nabbar/opendmarc-reports/config"
	"github.com/nabbar/opendmarc-reports/database"
	. "github.com/nabbar/opendmarc-reports/logger"
)

/*
Copyright 2018 Nicolas JUHEL

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

var reportCmd = &cobra.Command{
	Use:     "report",
	Example: "report",
	Short:   "Generate a report and send it",
	Long: `Load OpenDMARC history data from mysql database, 
generate report for selected domains or all domains, 
and sent it by mail through SMTP server.
`,
	Run: func(cmd *cobra.Command, args []string) {
		DebugLevel.LogData("Viper Settings : ", viper.AllSettings())

		config.GetConfig().Connect()
		database.CheckTables()

		lst, err := database.GetDomainList(false, config.GetConfig().IsDayMode(), config.GetConfig().GetInterval())
		FatalLevel.LogErrorCtx(NilLevel, "retrieve domain list to generate report", err)

		var wg sync.WaitGroup

		for _, dom := range lst {
			wg.Add(1)
			go GoRunDomain(&wg, dom)
		}

		DebugLevel.Logf("Waiting all threads finish...")
		wg.Wait()
		DebugLevel.Logf("All threads has finished")
	},
	Args: cobra.NoArgs,
}

func init() {
	rootCmd.AddCommand(reportCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func GoRunDomain(wg *sync.WaitGroup, id int) {
	//	var swg sync.WaitGroup
	defer func() {
		if wg != nil {
			wg.Done()
		}
		if r := recover(); r != nil {
			InfoLevel.Logf("Recover Panic Value : %v", r)
			return
		}
	}()

	dom, err := database.GetDomain(id)
	FatalLevel.LogErrorCtx(NilLevel, fmt.Sprintf("retrieve domain ID '%d' to generate report", id), err)
	DebugLevel.Logf("Starting Report Thread for domain: %s (Id: %d)", dom.Name, dom.Id)

	lst, err := database.GetRequestList(dom, false, config.GetConfig().IsDayMode(), config.GetConfig().GetInterval())
	FatalLevel.LogErrorCtx(NilLevel, fmt.Sprintf("retrieve request list for domain '%s' (Id : %d) to generate report", dom.Name, dom.Id), err)

	for _, req := range lst {
		wg.Add(1)
		go GoRunRequest(wg, req)
	}
}

func GoRunRequest(wg *sync.WaitGroup, id int) {
	defer func() {
		if wg != nil {
			wg.Done()
		}
		if r := recover(); r != nil {
			InfoLevel.Logf("Recover Panic Value : %v", r)
			return
		}
	}()

	req, err := database.GetRequests(id)
	PanicLevel.LogErrorCtx(NilLevel, fmt.Sprintf("retrieve request ID '%d' to generate report", id), err)
	PanicLevel.LogErrorCtx(NilLevel, fmt.Sprintf("generating report for request '%s' (id : %d)", req.Repuri, req.Id), req.SendReport(
		config.GetConfig().GetOrg(),
		config.GetConfig().GetEmail().String(),
		config.GetConfig().IsUpdate(), false,
		config.GetConfig().IsDayMode(),
		config.GetConfig().GetInterval(),
	))
}
