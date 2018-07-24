package report

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"net/smtp"

	"github.com/nabbar/opendmarc-reports/config"
	. "github.com/nabbar/opendmarc-reports/logger"
	"github.com/nabbar/opendmarc-reports/tools"
	"github.com/nabbar/opendmarc-reports/version"
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

func (rep reportFile) SendMail() {
	var (
		to  = rep.GetUriEmail()
		rcp = config.GetConfig().GetMakeRecipient(to)
		frm = config.GetConfig().GetEmail()
		tmp = make([]byte, 0)

		bnd string
		err error
		wrt io.WriteCloser
		cli *smtp.Client
	)

	defer func() {
		if r := recover(); r != nil {
			InfoLevel.Logf("Recover Panic Value : %v", r)
		}
		if cli != nil {
			err = cli.Close()
			PanicLevel.LogErrorCtx(InfoLevel, "closing SMTP connection", err)
		}
	}()

	cli = config.GetConfig().GetSMTP().Client()

	if to.IsEmpty() {
		InfoLevel.Logf("Skip Mail => Domain '%s' / Request '%s' / ZipFileName : '%s'", rep.GetDomain(), rep.repuri, rep.zipFile)
		return
	}

	if config.GetConfig().IsTesting() {
		InfoLevel.Logf("Testing mode : send zip file '%s' to mail '%s' instead of mail '[%v]'", rep.GetZipName(), frm.String(), to)
		to = tools.NewListMailAddress()
		to.Add(frm)
		rcp = config.GetConfig().GetMakeRecipient(to)
	}

	if rep.zipFile.Len() < 1 {
		PanicLevel.LogErrorCtx(NilLevel, "generating xml report file", errors.New("buffer is empty"))
	}

	err = cli.Noop()
	PanicLevel.LogErrorCtx(InfoLevel, "checking SMTP connection is up", err)

	err = cli.Mail(frm.String())
	PanicLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("pushing from mail '%s' to smtp server", frm.String()), err)

	for _, adr := range rcp {
		err = cli.Rcpt(adr.String())
		PanicLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("pushing new rcpt mail '%s' to smtp server", adr.String()), err)
	}

	wrt, err = cli.Data()
	PanicLevel.LogErrorCtx(DebugLevel, "create the IOWriter to smtp server", err)

	writeHeader(wrt, "From", frm.String())
	writeHeader(wrt, "To", to.String())

	prt := strings.Split(frm.String(), "@")
	dom := prt[len(prt)-1]
	rid := fmt.Sprintf("<%s@%s>", tools.NewMailAddress(rep.GetReportId(), "").String(), dom)

	sub := fmt.Sprintf(
		"Report domain: %s submitter: %s report-id: %s",
		tools.NewMailAddress(rep.GetDomain(), "").AddressOnly(),
		frm.AddressOnly(),
		tools.NewMailAddress(rep.GetReportId(), "").AddressOnly(),
	)

	writeHeader(wrt, "Subject", sub)
	writeHeader(wrt, "X-Mailer", tools.NewMailAddress(version.GetHeader(), "").String())
	writeHeader(wrt, "Date", time.Now().Format(time.RFC1123Z))
	writeHeader(wrt, "Message-ID", rid)
	writeHeader(wrt, "Auto-Submitted", "auto-generated")
	writeHeader(wrt, "MIME-Version", "1.0")

	bnd, err = tools.GenerateBoundary()
	PanicLevel.LogErrorCtxf(DebugLevel, "generating boundary '%s'", err, bnd)
	bnd = bnd[:28]

	writeHeader(wrt, "Content-Type", fmt.Sprintf("multipart/mixed; boundary=\"%s\"", bnd))

	writeCRLF(wrt)
	err = writeString(wrt, fmt.Sprintf("--%s", bnd))
	PanicLevel.LogErrorCtx(DebugLevel, fmt.Sprintf("writing multipart boundary '%s' part mail contents to smtp server", bnd), err)
	writeCRLF(wrt)

	writeHeader(wrt, "Content-Type", "application/zip")
	writeHeader(wrt, "Content-Transfer-Encoding", "base64")
	writeHeader(wrt, "Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", rep.GetZipName()))
	writeCRLF(wrt)

	enc := make([]byte, base64.StdEncoding.EncodedLen(rep.zipFile.Len()))
	base64.StdEncoding.Encode(enc, rep.zipFile.Bytes())

	if len(enc) < 1 {
		PanicLevel.LogErrorCtx(NilLevel, "encoding xml report file", errors.New("encoded buffer is empty"))
	}

	// write base64 content in lines of up to 76 chars
	tmp = make([]byte, 0)
	for i, l := 0, len(enc); i < l; i++ {
		tmp = append(tmp, enc[i])

		if (i+1)%76 == 0 {
			_, err = wrt.Write(tmp)
			PanicLevel.LogErrorCtx(NilLevel, "writing 76 bytes of attachment to smtp server", err)

			tmp = make([]byte, 0)
			writeCRLF(wrt)
		}
	}

	if len(tmp) != 0 {
		_, err = wrt.Write(tmp)
		PanicLevel.LogErrorCtx(NilLevel, "writing 76 bytes of attachment to smtp server", err)

		tmp = make([]byte, 0)
		writeCRLF(wrt)
	}

	writeCRLF(wrt)
	writeCRLF(wrt)
	err = writeString(wrt, fmt.Sprintf("--%s--", bnd))
	PanicLevel.LogErrorCtx(NilLevel, fmt.Sprintf("writing end multipart boundary '%s' part mail contents to smtp server", bnd), err)
	writeCRLF(wrt)
	writeCRLF(wrt)

	err = cli.Noop()
	PanicLevel.LogErrorCtx(InfoLevel, "checking SMTP connection is up", err)
}

func writeString(w io.WriteCloser, str string) error {
	if w == nil {
		return errors.New("empty writer")
	}

	_, err := w.Write([]byte(str))

	return err
}

func writeHeader(w io.WriteCloser, head, value string) {
	if w == nil {
		PanicLevel.LogErrorCtxf(DebugLevel, "writing header '%s: %s' to writer", errors.New("empty writer"), head, value)
	}

	_, err := w.Write([]byte(fmt.Sprintf("%s: %s\r\n", head, value)))
	PanicLevel.LogErrorCtxf(DebugLevel, "writing header '%s: %s' to writer", err, head, value)
}

func writeCRLF(w io.WriteCloser) {
	if w == nil {
		PanicLevel.LogErrorCtxf(DebugLevel, "writing CRLF '\\r\\n' to writer", errors.New("empty writer"))
	}

	_, err := w.Write([]byte("\r\n"))
	PanicLevel.LogErrorCtxf(DebugLevel, "writing CRLF '\\r\\n' to writer", err)
}
