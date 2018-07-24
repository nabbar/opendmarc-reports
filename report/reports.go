package report

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"strconv"
	"strings"

	"compress/flate"
	"fmt"
	"io"
	"time"

	"net/url"

	"sync"

	"github.com/kennygrant/sanitize"
	. "github.com/nabbar/opendmarc-reports/logger"
	"github.com/nabbar/opendmarc-reports/tools"
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

type feedback struct {
	MetaData ReportMetadata `xml:"report_metadata"`
	Policy   ReportPolicy   `xml:"policy_published"`
	Record   []ReportRecord `xml:"record"`
}

type reportFile struct {
	repuri  []string
	xmlFile feedback
	xmlName string
	zipName string
	zipFile *bytes.Buffer
}

type Report interface {
	Buffer(headerXml bool) (*bytes.Buffer, error)
	String(headerXml bool) (string, error)
	Byte(headerXml bool) ([]byte, error)
	Zip() (*bytes.Buffer, error)
	SendReport()

	GetFileName() string
	GetZipName() string

	GetFromEmail() *tools.MailAddress
	GetFromOrg() string
	GetReportId() string
	GetDomain() string
	GetDateRange() (time.Time, time.Time)

	GetUriEmail() tools.ListMailAddress
	GetUriHttp() []string
	GetUriFtp() []string
	GetUriUnknown() []string
}

func GetReport(repuri string, meta ReportMetadata, policy ReportPolicy, record []ReportRecord) Report {
	email := strings.Split(meta.Email, "@")
	domain := email[len(email)-1]

	baseName := strings.Join([]string{
		sanitize.Name(domain),
		sanitize.Name(policy.Domain),
		sanitize.Name(strconv.FormatInt(int64(meta.DateRange.Begin), 10)),
		sanitize.Name(strconv.FormatInt(int64(meta.DateRange.End), 10)),
	}, "!")

	return &reportFile{
		repuri: strings.Split(repuri, ","),
		xmlFile: feedback{
			MetaData: meta,
			Policy:   policy,
			Record:   record,
		},
		xmlName: baseName + ".xml",
		zipName: baseName + ".zip",
	}
}

func (rep reportFile) GetFromEmail() *tools.MailAddress {
	return tools.MailAddressParser(rep.xmlFile.MetaData.Email)
}

func (rep reportFile) GetFromOrg() string {
	return rep.xmlFile.MetaData.OrgName
}

func (rep reportFile) GetReportId() string {
	return rep.xmlFile.MetaData.ReportId
}

func (rep reportFile) GetDomain() string {
	return rep.xmlFile.Policy.Domain
}

func (rep reportFile) GetDateRange() (time.Time, time.Time) {
	return time.Unix(int64(rep.xmlFile.MetaData.DateRange.Begin), 0), time.Unix(int64(rep.xmlFile.MetaData.DateRange.End), 0)
}

func (rep reportFile) Buffer(headerXml bool) (*bytes.Buffer, error) {
	var (
		out = bytes.NewBuffer(make([]byte, 0))
	)

	if headerXml {
		out.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\" ?>\n")
	}

	if buf, err := xml.MarshalIndent(rep.xmlFile, "", "  "); err != nil {
		return nil, err
	} else if _, err = out.Write(buf); err != nil {
		return nil, err
	}

	return out, nil
}

func (rep reportFile) String(headerXml bool) (string, error) {
	buf, err := rep.Buffer(headerXml)

	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (rep reportFile) Byte(headerXml bool) ([]byte, error) {
	buf, err := rep.Buffer(headerXml)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (rep *reportFile) Zip() (*bytes.Buffer, error) {
	if rep.zipFile == nil || rep.zipFile.Len() < 1 {
		var (
			buf = bytes.NewBuffer(make([]byte, 0))
			wrt = zip.NewWriter(buf)
		)

		// Register a custom Deflate compressor.
		wrt.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
			return flate.NewWriter(out, flate.BestCompression)
		})

		wrt.SetComment(
			fmt.Sprintf("From: %s | Org: %s | Domain : %s | Date From : %s | Date To : %s",
				rep.xmlFile.MetaData.Email,
				rep.xmlFile.MetaData.OrgName,
				rep.xmlFile.Policy.Domain,
				time.Unix(int64(rep.xmlFile.MetaData.DateRange.Begin), 0).String(),
				time.Unix(int64(rep.xmlFile.MetaData.DateRange.End), 0).String(),
			),
		)

		if src, err := rep.Buffer(true); err != nil {
			return nil, err
		} else if f, err := wrt.Create(rep.xmlName); err != nil {
			return nil, err
		} else if _, err := f.Write(src.Bytes()); err != nil {
			return nil, err
		} else if err := wrt.Close(); err != nil {
			return nil, err
		}

		rep.zipFile = buf
	}

	return rep.zipFile, nil
}

func (rep reportFile) GetFileName() string {
	return rep.xmlName
}

func (rep reportFile) GetZipName() string {
	return rep.zipName
}

func (rep reportFile) parseUri(filter string) []string {
	var res = make([]string, 0)

	for _, s := range rep.repuri {
		if s == "" {
			continue
		}

		if u, e := url.Parse(s); e != nil {
			WarnLevel.LogErrorCtx(NilLevel, fmt.Sprintf("parsing rua list with value '%s'", s), e)
			continue
		} else if strings.HasPrefix(u.Scheme, filter) {
			res = append(res, strings.TrimSpace(s))
		}
	}

	return res
}

func (rep reportFile) isUriEmpty() bool {
	for _, s := range rep.repuri {
		if s != "-" {
			return false
		}
	}

	return true
}

func (rep reportFile) GetUriEmail() tools.ListMailAddress {
	var res = tools.NewListMailAddress()

	for _, adr := range rep.parseUri("mail") {
		if adr == "" {
			continue
		}

		t := strings.Split(adr, ":")
		m := strings.Join(t[1:], ":")
		res.AddParseEmail(m)
	}

	return res
}

func (rep reportFile) GetUriHttp() []string {
	return rep.parseUri("http")
}

func (rep reportFile) GetUriFtp() []string {
	return rep.parseUri("ftp")
}

func (rep reportFile) GetUriUnknown() []string {
	var res = make([]string, 0)

	for _, s := range rep.repuri {
		if s == "" {
			continue
		}

		if u, e := url.Parse(s); e != nil {
			res = append(res, s)
		} else if strings.HasPrefix(u.Scheme, "mail") {
			continue
		} else if !strings.HasPrefix(u.Scheme, "http") {
			continue
		} else if !strings.HasPrefix(u.Scheme, "ftp") {
			continue
		} else if s == "-" {
			continue
		} else {
			res = append(res, s)
		}
	}

	return res
}

func (rep *reportFile) SendReport() {
	if rep == nil || rep.isUriEmpty() {
		PanicLevel.LogErrorCtx(DebugLevel, "checking report", errors.New("empty or nil report"))
		return
	}

	_, err := rep.Zip()

	if err != nil || rep.zipFile.Len() < 1 {
		ErrorLevel.LogErrorCtx(NilLevel, "creating Zip file", err)
		return
	}

	var wg sync.WaitGroup

	for _, h := range rep.GetUriHttp() {
		wg.Add(1)
		go func() {
			rep.SendHttp(h)
			wg.Done()
		}()
	}

	for _, f := range rep.GetUriFtp() {
		wg.Add(1)
		go func() {
			rep.SendFtp(f)
			wg.Done()
		}()
	}

	for _, u := range rep.GetUriUnknown() {
		ErrorLevel.LogErrorCtx(NilLevel, fmt.Sprintf("parsing rua field with item '%s'", u), errors.New("unknown send method"))
	}

	rep.SendMail()
	InfoLevel.Logf("Waiting sending reports threads has finished...")
	wg.Wait()
}
