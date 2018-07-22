package database

import (
	"database/sql"
	"time"

	"fmt"

	"strings"

	"strconv"

	. "github.com/nabbar/opendmarc-reports/logger"
	"github.com/nabbar/opendmarc-reports/report"
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

const table_messages = "messages"
const field_messages = "`id`, `date`, `jobid`, `reporter`, `ip`, `policy`, `disp`, `from_domain`, `env_domain`, `policy_domain`, `sigcount`, `spf`, `align_spf`, `align_dkim`, `request_id`, `sent`"

type Messages struct {
	Generic

	Id           int
	Date         time.Time
	JobId        string
	Reporter     *Reporters
	Ip           *IpAddr
	Policy       int
	Disp         int
	FromDomain   *Domain
	EnvDomain    *Domain
	PolicyDomain *Domain
	SigCount     int
	SPF          int
	AlignSPF     int
	AlignDKIM    int
	Request      *Requests
	Sent         bool
}

func NewMessages(JobId string) *Messages {
	obj := &Messages{
		Generic: Generic{
			table: table_messages,
			fctField: func() FieldList {
				return FieldList{
					"id":            "int(11) NOT NULL AUTO_INCREMENT",
					"date":          "timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP",
					"jobid":         "varchar(128) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''",
					"reporter":      "int(10) unsigned NOT NULL DEFAULT '0'",
					"ip":            "int(10) unsigned NOT NULL DEFAULT '0'",
					"policy":        "tinyint(3) unsigned NOT NULL DEFAULT '0'",
					"disp":          "tinyint(3) unsigned NOT NULL DEFAULT '0'",
					"from_domain":   "int(10) unsigned NOT NULL DEFAULT '0'",
					"env_domain":    "int(10) unsigned NOT NULL DEFAULT '0'",
					"policy_domain": "int(10) unsigned NOT NULL DEFAULT '0'",
					"sigcount":      "tinyint(3) unsigned NOT NULL DEFAULT '0'",
					"spf":           "tinyint(3) unsigned NOT NULL DEFAULT '0'",
					"align_spf":     "tinyint(3) unsigned NOT NULL DEFAULT '0'",
					"align_dkim":    "tinyint(3) unsigned NOT NULL DEFAULT '0'",
					"request_id":    "int(10) unsigned NOT NULL DEFAULT '0'",
					"sent":          "tinyint(1) unsigned NOT NULL DEFAULT '0'",
				}
			},
			fctIndex: func() IndexList {
				return IndexList{
					"PRIMARY": {"type": "PRIMARY", "fields": "id"},
					"jobid":   {"type": "UNIQUE", "fields": "id,jobid"},
					"sent":    {"type": "UNIQUE", "fields": "id,date,from_domain,request_id,sent"},
				}
			},
		},
		Sent: false,
	}

	if JobId != "" {
		obj.JobId = JobId
	}

	return obj
}

func GetMessages(Id int) (*Messages, error) {
	obj := NewMessages("")

	var err error

	if Id != 0 {
		obj.Id = Id
		err = obj.Load()
	}

	return obj, err
}

func GetAllMessages(request *Requests, sent, dateMode bool, dateInterval time.Duration) (lst []*Messages, err error) {
	var rows *sql.Rows

	lst = make([]*Messages, 0)

	qry := fmt.Sprintf("SELECT %s FROM `%s`", field_messages, table_messages)
	arg := []interface{}{request.Id, sent}

	if dateMode {
		qry = qry + " WHERE `request_id`=? AND `sent`=? AND DATE(`date`) < DATE(DATE_SUB(CURRENT_DATE(), INTERVAL 1 DAY))"
	} else {
		qry = qry + " WHERE `request_id`=? AND `sent`=? AND `date` < DATE_SUB(NOW(), INTERVAL ? SECOND)"
		arg = append(arg, dateInterval.Seconds())
	}

	if rows, err = GetDbCli().Query(qry, arg...); err != nil {
		return
	} else if err = rows.Err(); err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		var (
			obj = NewMessages("")
		)

		if err = obj.parseRow(rows); err != nil {
			return
		} else if err = rows.Err(); err != nil {
			return
		}

		lst = append(lst, obj)
	}

	if err = rows.Err(); err != nil {
		return
	}

	return
}

func GetRangeDate(request *Requests, sent, dateMode bool, dateInterval time.Duration) (dateMin, dateMax int, err error) {
	var rows *sql.Rows

	qry := fmt.Sprintf("SELECT UNIX_TIMESTAMP(MIN(`date`)), UNIX_TIMESTAMP(MAX(`date`)) FROM `%s`", table_messages)
	arg := []interface{}{request.Id, sent}

	if dateMode {
		qry = qry + " WHERE `request_id`=? AND `sent`=? AND DATE(`date`) < DATE(DATE_SUB(CURRENT_DATE(), INTERVAL 1 DAY))"
	} else {
		qry = qry + " WHERE `request_id`=? AND `sent`=? AND `date` < DATE_SUB(NOW(), INTERVAL ? SECOND)"
		arg = append(arg, dateInterval.Seconds())
	}

	if rows, err = GetDbCli().Query(qry, arg...); err != nil {
		return
	} else if err = rows.Err(); err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&dateMin, &dateMax); err != nil {
			return
		} else if err = rows.Err(); err != nil {
			return
		}

		DebugLevel.Logf("Find date range into table %s : %s - %s", table_messages, time.Unix(int64(dateMin), 0).String(), time.Unix(int64(dateMax), 0).String())
		return
	}

	if err = rows.Err(); err != nil {
		return
	}

	return
}

func GetDomainList(sent, dateMode bool, dateInterval time.Duration) (domainIds []int, err error) {
	var rows *sql.Rows
	domainIds = make([]int, 0)

	qry := fmt.Sprintf("SELECT DISTINCT `from_domain` FROM `%s`", table_messages)
	arg := []interface{}{sent}

	if dateMode {
		qry = qry + " WHERE `sent`=? AND DATE(`date`) < DATE(DATE_SUB(CURRENT_DATE(), INTERVAL 1 DAY))"
	} else {
		qry = qry + " WHERE `sent`=? AND `date` < DATE_SUB(NOW(), INTERVAL ? SECOND)"
		arg = append(arg, dateInterval.Seconds())
	}

	if rows, err = GetDbCli().Query(qry, arg...); err != nil {
		return
	} else if err = rows.Err(); err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		var dom int

		if err = rows.Scan(&dom); err != nil {
			return
		} else if err = rows.Err(); err != nil {
			return
		}

		domainIds = append(domainIds, dom)
	}

	if err = rows.Err(); err != nil {
		return
	}

	return
}

func GetRequestList(domain *Domain, sent, dateMode bool, dateInterval time.Duration) (requestIds []int, err error) {
	var rows *sql.Rows
	requestIds = make([]int, 0)

	qry := fmt.Sprintf("SELECT DISTINCT `request_id` FROM `%s`", table_messages)
	arg := []interface{}{domain.Id, sent}

	if dateMode {
		qry = qry + " WHERE `from_domain`=? AND `sent`=? AND DATE(`date`) < DATE(DATE_SUB(CURRENT_DATE(), INTERVAL 1 DAY))"
	} else {
		qry = qry + " WHERE `from_domain`=? AND `sent`=? AND `date` < DATE_SUB(NOW(), INTERVAL ? SECOND)"
		arg = append(arg, dateInterval.Seconds())
	}

	if rows, err = GetDbCli().Query(qry, arg...); err != nil {
		return
	} else if err = rows.Err(); err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		var req int

		if err = rows.Scan(&req); err != nil {
			return
		} else if err = rows.Err(); err != nil {
			return
		}

		requestIds = append(requestIds, req)
	}

	if err = rows.Err(); err != nil {
		return
	}

	return
}

func (obj *Messages) parseRow(row *sql.Rows) error {
	var (
		rep int
		ipa int
		frm int
		env int
		pol int
		req int
	)

	err := row.Scan(
		&obj.Id,
		&obj.Date,
		&obj.JobId,
		&rep,
		&ipa,
		&obj.Policy,
		&obj.Disp,
		&frm,
		&env,
		&pol,
		&obj.SigCount,
		&obj.SPF,
		&obj.AlignSPF,
		&obj.AlignDKIM,
		&req,
		&obj.Sent,
	)

	if err != nil {
		return err
	} else if err = row.Err(); err != nil {
		return err
	}

	if rep > 0 {
		obj.Reporter, _ = GetReporters(rep)
	}
	if obj.Reporter == nil {
		obj.Reporter = NewReporters("")
	}

	if ipa > 0 {
		obj.Ip, _ = GetIpAddr(ipa)
	}
	if obj.Ip == nil {
		obj.Ip = NewIpAddr("")
	}

	if frm > 0 {
		obj.FromDomain, _ = GetDomain(frm)
	}
	if obj.FromDomain == nil {
		obj.FromDomain = NewDomain("")
	}

	if env > 0 {
		if env == obj.FromDomain.Id {
			obj.EnvDomain = obj.FromDomain
		} else {
			obj.EnvDomain, _ = GetDomain(env)
		}
	}
	if obj.EnvDomain == nil {
		obj.EnvDomain = NewDomain("")
	}

	if pol > 0 {
		if pol == obj.FromDomain.Id {
			obj.PolicyDomain = obj.FromDomain
		} else if pol == obj.EnvDomain.Id {
			obj.PolicyDomain = obj.EnvDomain
		} else {
			obj.PolicyDomain, _ = GetDomain(pol)
		}
	}
	if obj.PolicyDomain == nil {
		obj.PolicyDomain = NewDomain("")
	}

	DebugLevel.Logf("Find row into table %s : %s (id: %d)", obj.table, obj.JobId, obj.Id)
	return nil
}

func (obj *Messages) Load() error {
	var (
		rows *sql.Rows
		err  error
		qry  = fmt.Sprintf("SELECT %s FROM %s", field_messages, obj.table)
		arg  = make([]interface{}, 0)
	)

	if obj.Id != 0 {
		qry = qry + " WHERE `id`=? LIMIT 1"
		arg = []interface{}{obj.Id}
	} else if obj.JobId != "" {
		qry = qry + " WHERE `jobid`=? LIMIT 1"
		arg = []interface{}{obj.JobId}
	} else {
		return fmt.Errorf("cannot load null row into table %s", obj.table)
	}

	if rows, err = GetDbCli().Query(qry, arg...); err != nil {
		return err
	} else if err = rows.Err(); err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		if err = obj.parseRow(rows); err != nil {
			return err
		} else if err = rows.Err(); err != nil {
			return err
		}
		break
	}

	if err = rows.Err(); err != nil {
		return err
	}

	return nil
}

func (obj *Messages) Save() error {
	var (
		res sql.Result
		row int64
		nbr int64
		err error
	)

	if obj.Reporter != nil {
		err = obj.Reporter.Save()
		FatalLevel.LogErrorCtx(true, fmt.Sprintf("while saving Reporter for job '%s'", obj.JobId), err)
	} else {
		obj.Reporter = NewReporters("")
	}

	if obj.Ip != nil {
		err = obj.Ip.Save()
		FatalLevel.LogErrorCtx(true, fmt.Sprintf("while saving Ip for job '%s'", obj.JobId), err)
	} else {
		obj.Ip = NewIpAddr("")
	}

	if obj.FromDomain != nil {
		err = obj.FromDomain.Save()
		FatalLevel.LogErrorCtx(true, fmt.Sprintf("while saving From Domain for job '%s'", obj.JobId), err)
	} else {
		obj.FromDomain = NewDomain("")
	}

	if obj.EnvDomain != nil {
		err = obj.EnvDomain.Save()
		FatalLevel.LogErrorCtx(true, fmt.Sprintf("while saving Env Domain for job '%s'", obj.JobId), err)
	} else {
		obj.EnvDomain = NewDomain("")
	}

	if obj.PolicyDomain != nil {
		err = obj.PolicyDomain.Save()
		FatalLevel.LogErrorCtx(true, fmt.Sprintf("while saving Policy Domain for job '%s'", obj.JobId), err)
	} else {
		obj.PolicyDomain = NewDomain("")
	}

	if obj.Request != nil {
		req := NewRequests(obj.FromDomain)
		err = req.Load()
		FatalLevel.LogErrorCtx(true, fmt.Sprintf("while saving request for job '%s'", obj.JobId), err)

		if req.IsLocked() {
			return fmt.Errorf("request id '%d' for domain '%s' (id: %d) is locked", req.Id, req.Domain.Name, req.Domain.Id)
		}

		req.Repuri = tools.CleanJoin(tools.UnicSliceString(tools.CleanMergeSlice(strings.Split(req.Repuri, ","), strings.Split(obj.Request.Repuri, ",")...)), ",")
		if !obj.Request.Date.IsZero() {
			req.SetDateTime(obj.Request.Date)
		}

		if obj.Request.Pct != 0 {
			req.Pct = obj.Request.Pct
		}

		if obj.Request.Policy != 0 {
			req.Policy = obj.Request.Policy
		}

		if obj.Request.Spolicy != 0 {
			req.Spolicy = obj.Request.Spolicy
		}

		if obj.Request.ADKIM != 0 {
			req.ADKIM = obj.Request.ADKIM
		}

		if obj.Request.ASPF != 0 {
			req.ASPF = obj.Request.ASPF
		}

		obj.Request = req
		err = obj.Request.Save()
		FatalLevel.LogErrorCtx(true, fmt.Sprintf("while saving Request for job '%s'", obj.JobId), err)
	} else {
		obj.Request = NewRequests(obj.FromDomain)
		obj.Load()
	}

	if obj.Id != 0 {
		return obj.Update()
	}

	if obj.JobId == "" {
		return fmt.Errorf("cannot add an empty row into table %s", obj.table)
	}

	if obj.Date.IsZero() {
		obj.Date = time.Now()
	}

	fld := strings.SplitN(field_messages, ",", 2)
	lst := strings.TrimSpace(fld[1])

	res, err = GetDbCli().Exec(
		fmt.Sprintf("INSERT INTO `%s`(%s) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)", obj.table, lst),
		obj.Date,
		obj.JobId,
		obj.Reporter.Id,
		obj.Ip.Id,
		obj.Policy,
		obj.Disp,
		obj.FromDomain.Id,
		obj.EnvDomain.Id,
		obj.PolicyDomain.Id,
		obj.SigCount,
		obj.SPF,
		obj.AlignSPF,
		obj.AlignDKIM,
		obj.Request.Id,
		obj.Sent,
	)

	if err != nil {
		return err
	}

	row, err = res.RowsAffected()

	if err != nil {
		return err
	}

	if row != 0 {

		nbr, err = res.LastInsertId()

		if err != nil {
			return err
		}

		obj.Id = int(nbr)
		DebugLevel.Logf("Added %d row into table %s : %s (id: %d)", row, obj.table, obj.JobId, obj.Id)
	}

	return nil
}

func (obj *Messages) Update() error {
	var (
		res sql.Result
		row int64
		err error
	)

	if obj.Id == 0 {
		return obj.Save()
	}

	if obj.JobId == "" {
		return fmt.Errorf("cannot update an empty row into table %s", obj.table)
	}

	if obj.Date.IsZero() {
		obj.Date = time.Now()
	}

	sql := fmt.Sprintf("UPDATE `%s` SET `date` = ?, `jobid` = ?", obj.table)
	arg := []interface{}{obj.Date, obj.JobId}

	if obj.Reporter.Id != 0 {
		sql = sql + ", `reporter` = ? "
		arg = append(arg, obj.Reporter.Id)
	}

	if obj.Ip.Id != 0 {
		sql = sql + ", `ip` = ? "
		arg = append(arg, obj.Ip.Id)
	}

	if obj.Policy != 0 {
		sql = sql + ", `policy` = ? "
		arg = append(arg, obj.Policy)
	}

	if obj.Disp != 0 {
		sql = sql + ", `disp` = ? "
		arg = append(arg, obj.Disp)
	}

	if obj.FromDomain.Id != 0 {
		sql = sql + ", `from_domain` = ? "
		arg = append(arg, obj.FromDomain.Id)
	}

	if obj.EnvDomain.Id != 0 {
		sql = sql + ", `env_domain` = ? "
		arg = append(arg, obj.EnvDomain.Id)
	}

	if obj.PolicyDomain.Id != 0 {
		sql = sql + ", `policy_domain` = ? "
		arg = append(arg, obj.PolicyDomain.Id)
	}

	if obj.SigCount != 0 {
		sql = sql + ", `sigcount` = ? "
		arg = append(arg, obj.SigCount)
	}

	if obj.SPF != 0 {
		sql = sql + ", `spf` = ? "
		arg = append(arg, obj.SPF)
	}

	if obj.AlignSPF != 0 {
		sql = sql + ", `align_spf` = ? "
		arg = append(arg, obj.AlignSPF)
	}

	if obj.AlignDKIM != 0 {
		sql = sql + ", `align_dkim` = ? "
		arg = append(arg, obj.AlignDKIM)
	}

	if obj.Request.Id != 0 {
		sql = sql + ", `request_id` = ? "
		arg = append(arg, obj.Request.Id)
	}

	sql = sql + ", `sent` = ? "
	arg = append(arg, obj.Sent)

	arg = append(arg, obj.Id)
	res, err = GetDbCli().Exec(sql+" WHERE `id`=? LIMIT 1", arg...)

	if err != nil {
		return err
	}

	row, err = res.RowsAffected()

	if err != nil {
		return err
	}

	if row != 0 {
		DebugLevel.Logf("Updated %d row into table %s : %s (id: %d)", row, obj.table, obj.JobId, obj.Id)
	}

	return nil
}

func (obj *Messages) Delete() error {
	var (
		res sql.Result
		row int64
		err error
	)

	if obj.Id == 0 {
		return fmt.Errorf("cannot delete an empty or not saved row into table %s", obj.table)
	}

	res, err = GetDbCli().Exec(fmt.Sprintf("DELETE FROM `%s` WHERE `id`=? LIMIT 1", obj.table), obj.Id)

	if err != nil {
		return err
	}

	row, err = res.RowsAffected()

	if err != nil {
		return err
	}

	if row != 0 {
		DebugLevel.Logf("Deleted %d row into table %s : %s (id: %d)", row, obj.table, obj.JobId, obj.Id)
	}

	return nil
}

func (obj *Messages) SetDate(received string) error {
	var (
		val time.Time
		nix int64
		err error
	)

	if nix, err = strconv.ParseInt(received, 10, 64); err == nil {
		obj.Date = time.Unix(nix, 0)
		return nil
	}

	if val, err = time.Parse("2018-07-14T22:08:13+02:00", received); err != nil {
		return err
	}

	obj.Date = val
	return nil
}

func (obj *Messages) SetReporter(reporter string) error {
	var (
		sub *Reporters
		err error
	)

	sub = NewReporters(reporter)

	if err = sub.Load(); err != nil {
		return err
	}

	if err = sub.Save(); err != nil {
		return err
	}

	obj.Reporter = sub
	return nil
}

func (obj *Messages) SetIpAddr(ipaddr string) error {
	var (
		sub *IpAddr
		err error
	)

	sub = NewIpAddr(ipaddr)

	if err = sub.Load(); err != nil {
		return err
	}

	if err = sub.Save(); err != nil {
		return err
	}

	obj.Ip = sub
	return nil
}

func (obj *Messages) SetFromDomain(fromDomain string) error {
	var (
		sub *Domain
		err error
	)

	sub = NewDomain(fromDomain)

	if err = sub.Load(); err != nil {
		return err
	}

	if err = sub.Save(); err != nil {
		return err
	}

	obj.FromDomain = sub
	return nil
}

func (obj *Messages) SetEnvDomain(envDomain string) error {
	var (
		sub *Domain
		err error
	)

	sub = NewDomain(envDomain)

	if err = sub.Load(); err != nil {
		return err
	}

	if err = sub.Save(); err != nil {
		return err
	}

	obj.EnvDomain = sub
	return nil
}

func (obj *Messages) SetPolicyDomain(polDomain string) error {
	var (
		sub *Domain
		err error
	)

	sub = NewDomain(polDomain)

	if err = sub.Load(); err != nil {
		return err
	}

	if err = sub.Save(); err != nil {
		return err
	}

	obj.PolicyDomain = sub
	return nil
}

func (obj *Messages) GetSignatures() ([]*Signatures, error) {
	return GetAllSignatures(obj)
}

func (obj *Messages) GetDkimReport() ([]report.ReportDKIM, error) {
	if lst, err := GetAllSignatures(obj); err != nil {
		return nil, err
	} else {
		var res = make([]report.ReportDKIM, 0)
		for _, s := range lst {
			if s != nil && s.Id != 0 {
				res = append(res, s.GetReport())
			}
		}
		return res, nil
	}
}

func (obj Messages) GetReport() (report.ReportRecord, error) {
	var (
		lst []report.ReportDKIM
		err error
	)

	if lst, err = obj.GetDkimReport(); err != nil {
		return report.ReportRecord{}, err
	}

	return report.GetReportRecord(obj.Ip.Name, obj.GetDisp(), obj.GetAlignDKIM(), obj.GetAlignSPF(), obj.FromDomain.Name, obj.EnvDomain.Name, obj.GetSPF(), 1, lst), nil
}

func (obj *Messages) GetDisp() string {
	switch obj.Disp {
	case 0:
		return "reject"
	case 1:
		return "reject"
	case 2:
		return "none"
	case 4:
		return "quarantine"
	default:
		return "unknown"

	}
}

func (obj *Messages) GetSPF() string {
	switch obj.SPF {
	case 0:
		return "pass"
	case 2:
		return "softfail"
	case 3:
		return "neutral"
	case 4:
		return "temperror"
	case 5:
		return "permerror"
	case 6:
		return "none"
	case 7:
		return "fail"
	case 8:
		return "policy"
	case 9:
		return "nxdomain"
	case 10:
		return "signed"
	case 12:
		return "discard"
	default:
		return "unknown"

	}
}

func (obj *Messages) GetAlignDKIM() string {
	switch obj.AlignDKIM {
	case 4:
		return "pass"
	case 5:
		return "fail"
	default:
		return "unknown"

	}
}

func (obj *Messages) GetAlignSPF() string {
	switch obj.AlignSPF {
	case 4:
		return "pass"
	case 5:
		return "fail"
	default:
		return "unknown"

	}
}
