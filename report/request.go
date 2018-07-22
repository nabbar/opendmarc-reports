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
type ReportMetadata struct {
	OrgName   string `xml:"org_name"`
	Email     string `xml:"email"`
	ReportId  string `xml:"report_id"`
	DateRange struct {
		Begin int `xml:"begin"`
		End   int `xml:"end"`
	} `xml:"date_range"`
}

func GetReportMetadata(org, email string, id string, begin, end int) ReportMetadata {
	return ReportMetadata{
		OrgName:  org,
		Email:    email,
		ReportId: id,
		DateRange: struct {
			Begin int `xml:"begin"`
			End   int `xml:"end"`
		}{
			Begin: begin,
			End:   end,
		},
	}
}

type ReportPolicy struct {
	Domain string `xml:"domain"`
	ADKIM  string `xml:"adkim"`
	ASPF   string `xml:"aspf"`
	P      string `xml:"p"`
	SP     string `xml:"sp,omitempty"`
	PCT    int    `xml:"pct"`
}

func GetReportPolicy(domain, adkim, aspf, p, sp string, pct int) ReportPolicy {
	return ReportPolicy{
		Domain: domain,
		ADKIM:  adkim,
		ASPF:   adkim,
		P:      p,
		SP:     sp,
		PCT:    pct,
	}
}
