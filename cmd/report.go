package cmd

import (
	"fmt"
	"sync"

	"bytes"
	"errors"
	"strings"

	"time"

	"io"

	"encoding/base64"

	"github.com/nabbar/opendmarc-reports/config"
	"github.com/nabbar/opendmarc-reports/database"
	. "github.com/nabbar/opendmarc-reports/logger"
	"github.com/nabbar/opendmarc-reports/report"
	"github.com/nabbar/opendmarc-reports/tools"
	"github.com/nabbar/opendmarc-reports/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		FatalLevel.LogErrorCtx(false, "retrieve domain list to generate report", err)

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

	dom, err := database.GetDomain(id)
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("retrieve domain ID '%d' to generate report", id), err)
	DebugLevel.Logf("Starting Report Thread for domain: %s (Id: %d)", dom.Name, dom.Id)

	lst, err := database.GetRequestList(dom, false, config.GetConfig().IsDayMode(), config.GetConfig().GetInterval())
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("retrieve request list for domain '%s' (Id : %d) to generate report", dom.Name, dom.Id), err)

	for _, req := range lst {
		//		swg.Add(1)
		//		go GoRunRequest(&swg, req)
		wg.Add(1)
		go GoRunRequest(wg, req)
	}

	//	swg.Wait()
	wg.Done()
}

func GoRunRequest(wg *sync.WaitGroup, id int) {
	//	var swg sync.WaitGroup
	var buf *bytes.Buffer

	req, err := database.GetRequests(id)
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("retrieve request ID '%d' to generate report", id), err)
	DebugLevel.Logf("Starting Report Thread for request: %s (Id: %d)", req.Repuri, req.Id)

	rep, err := req.GetReport(
		config.GetConfig().GetOrg(),
		config.GetConfig().GetEmail().String(),
		config.GetConfig().IsUpdate(), false,
		config.GetConfig().IsDayMode(),
		config.GetConfig().GetInterval(),
	)
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("generating report for request '%s' (id : %d)", req.Repuri, req.Id), err)

	//	zipfile, zipbuf, err := rep.Zip()

	if err != nil {
		FatalLevel.LogErrorCtx(true, fmt.Sprintf("creating zip file for request '%s' (id : %d)", req.Repuri, req.Id), err)
	}

	buf, err = rep.Zip()
	FatalLevel.LogErrorCtx(false, "generating xml report file", err)

	for _, h := range rep.GetUriHttp() {
		//		swg.Add(1)
		//		go GoRunHttp(&swg, rep, h)
		wg.Add(1)
		go GoRunHttp(wg, rep, h, buf)
	}

	for _, f := range rep.GetUriFtp() {
		//		swg.Add(1)
		//		go GoRunFtp(&swg, rep, f)
		wg.Add(1)
		go GoRunFtp(wg, rep, f, buf)
	}

	for _, u := range rep.GetUriUnknown() {
		ErrorLevel.LogErrorCtx(true, fmt.Sprintf("parsing rua field with item '%s'", u), errors.New("unknown send method"))
	}

	//	swg.Wait()
	GoRunMail(wg, rep, buf)
	wg.Done()
}

func GoRunMail(wg *sync.WaitGroup, rep report.Report, zip *bytes.Buffer) {
	var (
		to  = rep.GetUriEmail()
		rcp = config.GetConfig().GetMakeRecipient(to)
		frm = config.GetConfig().GetEmail()
		tmp = make([]byte, 0)
		cli = config.GetConfig().GetSMTP().Client()

		bnd string
		err error
		wrt io.WriteCloser
	)

	DebugLevel.Logf("Starting Report Thread for email uri: %s", to)

	if to.IsEmpty() {
		return
	}

	if config.GetConfig().IsTesting() {
		InfoLevel.Logf("Testing mode : send zip file '%s' to mail '%s' instead of mail '[%v]'", rep.GetZipName(), frm.String(), to)
		to = tools.NewListMailAddress()
		to.Add(frm)
		rcp = config.GetConfig().GetMakeRecipient(to)
	}

	err = cli.Noop()
	FatalLevel.LogErrorCtx(false, "checking SMTP connection is up", err)
	InfoLevel.Logf("SMTP connection is opened")

	err = cli.Mail(frm.String())
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("pushing from mail '%s' to smtp server", frm.String()), err)

	for _, adr := range rcp {
		err = cli.Rcpt(adr.String())
		FatalLevel.LogErrorCtx(true, fmt.Sprintf("pushing new rcpt mail '%s' to smtp server", adr.String()), err)
	}

	wrt, err = cli.Data()
	FatalLevel.LogErrorCtx(false, "create the IOWriter to smtp server", err)

	err = writeString(wrt, fmt.Sprintf("%s: %s\r\n", "From", frm.String()))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("pushing from mail '%s' header to smtp server", frm.String()), err)

	err = writeString(wrt, fmt.Sprintf("%s: %s\r\n", "To", to.String()))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("pushing to mail '%s' header to smtp server", to.String()), err)

	prt := strings.Split(frm.String(), "@")
	dom := prt[len(prt)-1]
	rid := fmt.Sprintf("<%s@%s>", tools.NewMailAddress(rep.GetReportId(), "").String(), dom)

	sub := fmt.Sprintf(
		"%s: %s\r\n",
		"Subject",
		fmt.Sprintf(
			"Report domain: %s submitter: %s report-id: %s",
			tools.NewMailAddress(rep.GetDomain(), "").AddressOnly(),
			frm.AddressOnly(),
			tools.NewMailAddress(rep.GetReportId(), "").AddressOnly(),
		),
	)

	err = writeString(wrt, sub)
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing subject mail header '%s' to smtp server", sub), err)

	err = writeString(wrt, fmt.Sprintf("%s: %s\r\n", "X-Mailer", tools.NewMailAddress(version.GetHeader(), "").String()))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing x-mailer mail header '%s' to smtp server", version.GetHeader()), err)

	err = writeString(wrt, fmt.Sprintf("%s: %s\r\n", "Date", time.Now().Format(time.RFC1123Z)))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing date mail header '%s' to smtp server", time.Now().Format(time.RFC1123Z)), err)

	err = writeString(wrt, fmt.Sprintf("%s: %s\r\n", "Message-ID", rid))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing message-id mail header '%s' to smtp server", rid), err)

	err = writeString(wrt, fmt.Sprintf("%s: %s\r\n", "Auto-Submitted", "auto-generated"))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing auto-submitted mail header '%s' to smtp server", "auto-generated"), err)

	err = writeString(wrt, fmt.Sprintf("%s: %s\r\n", "MIME-Version", "1.0"))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing mime mail header '%s' to smtp server", "1.0"), err)

	bnd, err = tools.GenerateBoundary()
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing content-type mail header '%s' to smtp server", "auto-generated"), err)
	bnd = bnd[:28]

	err = writeString(wrt, fmt.Sprintf("%s: multipart/mixed; boundary=\"%s\"\r\n", "Content-Type", bnd))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing content-type mail header with boundary '%s' to smtp server", bnd), err)

	err = writeString(wrt, fmt.Sprintf("\r\n--%s\r\n", bnd))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing multipart boundary '%s' part mail contents to smtp server", bnd), err)

	err = writeString(wrt, fmt.Sprintf("%s: %s\r\n", "Content-Type", "application/xml"))
	FatalLevel.LogErrorCtx(true, "writing content-type zip mail header to smtp server", err)

	err = writeString(wrt, fmt.Sprintf("%s: %s\r\n", "Content-Transfer-Encoding", "base64"))
	FatalLevel.LogErrorCtx(true, "writing content-transfert mode mail header to smtp server", err)

	err = writeString(wrt, fmt.Sprintf("%s: attachment; filename=\"%s\"\r\n\r\n", "Content-Disposition", rep.GetZipName()))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing content-disposition with file name '%s' mail header to smtp server", rep.GetZipName()), err)

	if zip.Len() < 1 {
		FatalLevel.LogErrorCtx(false, "generating xml report file", errors.New("buffer is empty"))
	}

	enc := make([]byte, base64.StdEncoding.EncodedLen(zip.Len()))
	base64.StdEncoding.Encode(enc, zip.Bytes())

	if len(enc) < 1 {
		FatalLevel.LogErrorCtx(false, "encoding xml report file", errors.New("encoded buffer is empty"))
	}

	// write base64 content in lines of up to 76 chars
	tmp = make([]byte, 0)
	for i, l := 0, len(enc); i < l; i++ {
		tmp = append(tmp, enc[i])

		if (i+1)%76 == 0 {
			_, err = wrt.Write(tmp)
			FatalLevel.LogErrorCtx(false, "writing 76 bytes of attachment to smtp server", err)

			tmp = make([]byte, 0)

			_, err = wrt.Write([]byte("\r\n"))
			FatalLevel.LogErrorCtx(false, "writing CRLF to smtp server", err)
		}
	}

	if len(tmp) != 0 {
		_, err = wrt.Write(tmp)
		FatalLevel.LogErrorCtx(false, "writing 76 bytes of attachment to smtp server", err)

		tmp = make([]byte, 0)

		_, err = wrt.Write([]byte("\r\n"))
		FatalLevel.LogErrorCtx(false, "writing CRLF to smtp server", err)
	}

	err = writeString(wrt, fmt.Sprintf("\r\n\r\n--%s--\r\n\r\n", bnd))
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("writing end multipart boundary '%s' part mail contents to smtp server", bnd), err)

	err = cli.Noop()
	FatalLevel.LogErrorCtx(false, "checking SMTP connection is up", err)

	err = cli.Close()
	FatalLevel.LogErrorCtx(false, "closing SMTP connection", err)
	InfoLevel.Logf("SMTP connection is closed")

	//wg.Done()
}

func GoRunHttp(wg *sync.WaitGroup, rep report.Report, uri string, zip *bytes.Buffer) {
	DebugLevel.Logf("Starting Report Thread for http uri: %s", uri)

	if config.GetConfig().IsTesting() {
		InfoLevel.Logf("Testing mode : don't post zip file '%s' to url '%s'", rep.GetZipName(), uri)
		wg.Done()
		return
	}

	cli := config.GetConfig().GetHTTP(uri)

	if cli.Check() {
		cli.Call(zip)
	}

	wg.Done()
}

func GoRunFtp(wg *sync.WaitGroup, rep report.Report, uri string, zip *bytes.Buffer) {
	DebugLevel.Logf("Starting Report Thread for ftp uri: %s", uri)

	if config.GetConfig().IsTesting() {
		InfoLevel.Logf("Testing mode : don't upload zip file '%s' to ftp '%s'", rep.GetZipName(), uri)
		wg.Done()
		return
	}

	cli := config.GetConfig().GetFTP(uri)
	defer cli.Close()
	cli.Store(rep.GetFileName(), zip)

	wg.Done()
}

func writeString(w io.WriteCloser, str string) error {
	if w == nil {
		return errors.New("empty writer")
	}

	_, err := w.Write([]byte(str))

	return err
}
