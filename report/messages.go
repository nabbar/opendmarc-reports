package report

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
type ReportRecord struct {
	Row         ReportRow `xml:"row"`
	Identifiers struct {
		HeaderFrom string `xml:"header_from"`
	} `xml:"identifiers"`
	AuthResults ReportAuth `xml:"auth_results"`
}

func GetReportRecord(ip, disp, dkim, spf, from, domain, result string, count int, repdkim []ReportDKIM) ReportRecord {
	return ReportRecord{
		Row: GetReportRow(ip, disp, dkim, spf, count),
		Identifiers: struct {
			HeaderFrom string `xml:"header_from"`
		}{
			HeaderFrom: from,
		},
		AuthResults: GetReportAuth(domain, result, repdkim),
	}
}

type ReportRow struct {
	SourceIp        string `xml:"source_ip"`
	Count           int    `xml:"count"`
	PolicyEvaluated struct {
		Disposition string `xml:"disposition"`
		DKIM        string `xml:"dkim"`
		SPF         string `xml:"spf"`
	} `xml:"policy_evaluated"`
}

func GetReportRow(ip, disp, dkim, spf string, count int) ReportRow {
	return ReportRow{
		SourceIp: ip,
		Count:    count,
		PolicyEvaluated: struct {
			Disposition string `xml:"disposition"`
			DKIM        string `xml:"dkim"`
			SPF         string `xml:"spf"`
		}{
			Disposition: disp,
			DKIM:        dkim,
			SPF:         spf,
		},
	}
}

type ReportAuth struct {
	SPF  ReportSPF    `xml:"spf"`
	DKIM []ReportDKIM `xml:"dkim"`
}

type ReportSPF struct {
	Domain string `xml:"domain"`
	Result string `xml:"result"`
}

func GetReportAuth(domain, result string, dkim []ReportDKIM) ReportAuth {
	return ReportAuth{
		SPF:  GetReportSPF(domain, result),
		DKIM: dkim,
	}
}

func GetReportSPF(domain, result string) ReportSPF {
	return ReportSPF{
		Domain: domain,
		Result: result,
	}
}
