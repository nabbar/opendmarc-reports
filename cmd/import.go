package cmd

import (
	"errors"

	"fmt"
	"path/filepath"

	"bufio"
	"os"
	"strings"

	"strconv"

	"sync"

	"github.com/nabbar/opendmarc-reports/config"
	"github.com/nabbar/opendmarc-reports/database"
	. "github.com/nabbar/opendmarc-reports/logger"
	"github.com/nabbar/opendmarc-reports/tools"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
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

// configCmd represents the config command
var importCmd = &cobra.Command{
	Use:     "import <dat file pattern> [<dat file pattern>, ...]",
	Example: "import /var/tmp/dmarc.dat /var/tmp/opendmarc.*",
	Short:   "Import dat history file",
	Long: `Import OpenDMARC history file
into mysql database. If not exist create 
the record else update it.
`,
	Run: func(cmd *cobra.Command, args []string) {
		DebugLevel.LogData("Viper Settings : ", viper.AllSettings())

		config.GetConfig().Connect()
		database.CheckTables()

		var wg sync.WaitGroup

		for k, a := range args {
			wg.Add(1)
			lst, _ := filepath.Glob(a)
			go parseFileList(&wg, k, lst)
		}

		DebugLevel.Logf("Waiting all threads finish...")
		wg.Wait()
		DebugLevel.Logf("All threads has finished")
	},
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("arguments missing : requires at least one file path pattern")
		}

		for _, a := range args {
			if _, err := filepath.Glob(a); err != nil {
				return fmt.Errorf("Argument '%s' error: %v", a, err)
			}
		}

		return nil
	},
}

type jobItem struct {
	database.Messages
	signature []*database.Signatures
}

func NewJobItem(jobId string) *jobItem {
	job := &jobItem{}
	msg := database.NewMessages(jobId)
	job.Messages = *msg

	if job.Request == nil {
		job.Request = database.NewRequests(nil)
	}

	if job.signature == nil {
		job.signature = make([]*database.Signatures, 0)
	}

	return job
}

func init() {
	rootCmd.AddCommand(importCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func parseFileList(wg *sync.WaitGroup, nbr int, fileList []string) {
	DebugLevel.Logf("Starting thread #%d for filelist", nbr)

	var swg sync.WaitGroup

	for k, f := range fileList {
		swg.Add(1)
		go parseFile(&swg, nbr, k, f)
	}

	swg.Wait()
	wg.Done()
}

func parseFile(wg *sync.WaitGroup, nbr, sub int, filepath string) {
	DebugLevel.Logf("Starting parsing thread #%d-#%d for file: %s", nbr, sub, filepath)
	InfoLevel.Logf("Parsing file: %s ...", filepath)

	_, e := os.Stat(filepath)
	if ErrorLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("checking file '%s'", filepath), e) {
		return
	}

	f, e := os.Open(filepath)
	if ErrorLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("opening file '%s'", filepath), e) {
		return
	}

	defer f.Close()

	s := bufio.NewScanner(f)
	if ErrorLevel.LogErrorCtx(NilLevel, fmt.Sprintf("reading file '%s'", filepath), s.Err()) {
		return
	}

	var j = NewJobItem("")
	InfoLevel.Logf("Parsing file '%s'...", filepath)

	for s.Scan() {
		if ErrorLevel.LogErrorCtx(NilLevel, fmt.Sprintf("reading file '%s'", filepath), s.Err()) {
			continue
		}

		p := strings.SplitN(s.Text(), " ", 2)

		if len(p) != 2 {
			ErrorLevel.LogErrorCtx(NilLevel, fmt.Sprintf("reading file '%s'", filepath), errors.New("line not well formatted !"))
			continue
		}

		var (
			val int64
			err error
		)

		switch strings.ToLower(p[0]) {
		case "job":
			if j.JobId != "" {
				InfoLevel.Logf("Saving Job Id '%s' from file %s...", j.JobId, filepath)
				j.SaveJob()
			}

			InfoLevel.Logf("New Job '%s' found in file '%s'...", p[1], filepath)

			j = NewJobItem(p[1])
			err = j.Load()
			WarnLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("loading data for job '%s'", j.JobId), err)

		case "action":
			if val, err = strconv.ParseInt(p[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'action' for job '%s'", j.JobId), err)
				j.Disp = 0
			} else {
				j.Disp = int(val)
			}

		case "adkim":
			if val, err = strconv.ParseInt(p[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'adkim' for job '%s'", j.JobId), err)
				j.Request.ADKIM = 0
			} else {
				j.Request.ADKIM = int(val)
			}

		case "align_dkim":
			if val, err = strconv.ParseInt(p[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'align_dkim' for job '%s'", j.JobId), err)
				j.AlignDKIM = 0
			} else {
				j.AlignDKIM = int(val)
			}

		case "align_spf":
			if val, err = strconv.ParseInt(p[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'align_spf' for job '%s'", j.JobId), err)
				j.AlignSPF = 0
			} else {
				j.AlignSPF = int(val)
			}

		case "aspf":
			if val, err = strconv.ParseInt(p[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'aspf' for job '%s'", j.JobId), err)
				j.Request.ASPF = 0
			} else {
				j.Request.ASPF = int(val)
			}

		case "dkim":
			sig := database.NewSignatures(nil)
			d := strings.SplitN(p[1], " ", 2)
			sig.Domain = database.NewDomain(d[0])
			err = sig.Domain.Load()
			WarnLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("loading value 'dkim domain' for job '%s'", j.JobId), err)

			if val, err = strconv.ParseInt(d[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'dkim result' for job '%s'", j.JobId), err)
				sig.Pass = 5
			} else {
				sig.Pass = int(val)
			}

			if sig.Pass == 4 || sig.Pass == 5 {
				sig.Error = true
			} else {
				sig.Error = false
			}

			j.signature = append(j.signature, sig)
			j.SigCount++
			DebugLevel.Logf("Find new signature for job id '%s', total sign : %d(%d)", j.JobId, len(j.signature), j.SigCount)

		case "from":
			err = j.SetFromDomain(p[1])
			WarnLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("loading domain from '%s' for job '%s'", p[1], j.JobId), err)

		case "ipaddr":
			err = j.SetIpAddr(p[1])
			WarnLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("loading ipaddr '%s' for job '%s'", p[1], j.JobId), err)

		case "mfrom":
			err = j.SetEnvDomain(p[1])
			WarnLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("loading domain env '%s' for job '%s'", p[1], j.JobId), err)

		case "p":
			if val, err = strconv.ParseInt(p[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'p' for job '%s'", j.JobId), err)
				j.Request.Policy = 0
			} else {
				j.Request.Policy = int(val)
			}

		case "pct":
			if val, err = strconv.ParseInt(p[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'pct' for job '%s'", j.JobId), err)
				j.Request.Pct = 0
			} else {
				j.Request.Pct = int(val)
			}

		case "pdomain":
			err = j.SetPolicyDomain(p[1])
			WarnLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("loading domain policy '%s' for job '%s'", p[1], j.JobId), err)

		case "policy":
			if val, err = strconv.ParseInt(p[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'policy' for job '%s'", j.JobId), err)
				j.Policy = 0
			} else {
				j.Policy = int(val)
			}

		case "received":
			err = j.SetDate(p[1])
			WarnLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("setting Date '%s' for job '%s'", p[1], j.JobId), err)

			err = j.Request.SetDate(p[1])
			WarnLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("setting Request Date '%s' for job '%s'", p[1], j.JobId), err)

		case "reporter":
			err = j.SetReporter(p[1])

		case "rua":
			if !j.Request.IsLocked() {
				j.Request.Repuri = tools.CleanJoin(tools.UnicSliceString(tools.CleanMergeSlice(strings.Split(j.Request.Repuri, ","), p[1])), ",")
			}

		case "sp":
			if val, err = strconv.ParseInt(p[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'policy' for job '%s'", j.JobId), err)
				j.Request.Spolicy = 0
			} else {
				j.Request.Spolicy = int(val)
			}

		case "spf":
			if val, err = strconv.ParseInt(p[1], 10, 64); err != nil {
				WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("converting value 'policy' for job '%s'", j.JobId), err)
				j.SPF = 0
			} else {
				j.SPF = int(val)
			}

		default:
			ErrorLevel.LogErrorCtx(NilLevel, fmt.Sprintf("reading file '%s'", filepath), fmt.Errorf("key '%s' not understand", p[0]))
		}
	}

	if j.JobId != "" {
		InfoLevel.Logf("Saving Job Id '%s' from file %s...", j.JobId, filepath)
		j.SaveJob()
	}

	wg.Done()
}

func (job jobItem) String() string {
	str, err := yaml.Marshal(job)
	ErrorLevel.LogErrorCtx(NilLevel, "yaml encoding job", err)

	return fmt.Sprintf("---\n%s\n", string(str))

}

func (job *jobItem) SaveJob() {
	if job == nil {
		FatalLevel.LogErrorCtx(NilLevel, fmt.Sprintf("saving job '%s'", job), errors.New("invalid job reference - segment fault"))
	}

	if err := job.Save(); err != nil {
		FatalLevel.LogErrorCtx(NilLevel, fmt.Sprintf("saving job '%s'", job), err)
		return
	}

	var ids = make([]int, 0)
	for k, _ := range job.signature {
		if job.signature[k] == nil {
			continue
		}

		job.signature[k].Message = &job.Messages

		if err := job.signature[k].Save(); err != nil {
			FatalLevel.LogErrorCtx(NilLevel, fmt.Sprintf("saving sign's job '%s'", job), err)
			return
		}

		ids = append(ids, job.signature[k].Id)
	}

	InfoLevel.Logf("Job Id '%s' saved (Msg %d, Req %d, Sig %v", job.JobId, job.Id, job.Request.Id, ids)
}
