package database

import (
	"database/sql"
	"time"

	"fmt"

	"strings"

	"strconv"

	"errors"

	. "github.com/nabbar/opendmarc-reports/logger"
	"github.com/nabbar/opendmarc-reports/report"
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

const table_requests = "requests"
const field_requests = "`id`, `date`, `domain`, `repuri`, `pct`, `policy`, `spolicy`, `aspf`, `adkim`, `locked`"

type Requests struct {
	Generic

	Domain  *Domain
	Repuri  string
	Pct     int
	Policy  int
	Spolicy int
	ASPF    int
	ADKIM   int
	Locked  bool
}

func NewRequests(domain *Domain) *Requests {
	obj := &Requests{
		Generic: Generic{
			table: table_requests,
			fctField: func() FieldList {
				return FieldList{
					"id":      "int(11) NOT NULL AUTO_INCREMENT",
					"date":    "timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP",
					"domain":  "int(11) NOT NULL DEFAULT '0'",
					"repuri":  "varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''",
					"pct":     "tinyint(4) NOT NULL DEFAULT '0'",
					"policy":  "tinyint(4) NOT NULL DEFAULT '0'",
					"spolicy": "tinyint(4) NOT NULL DEFAULT '0'",
					"aspf":    "tinyint(4) NOT NULL DEFAULT '0'",
					"adkim":   "tinyint(4) NOT NULL DEFAULT '0'",
					"locked":  "tinyint(4) NOT NULL DEFAULT '0'",
				}
			},
			fctIndex: func() IndexList {
				return IndexList{
					"PRIMARY": {"type": "PRIMARY", "fields": "id"},
					"domain":  {"type": "UNIQUE", "fields": "id,domain"},
				}
			},
		},
	}

	if domain != nil && domain.Id != 0 {
		obj.Domain = domain
	} else {
		obj.Domain = NewDomain("")
	}

	return obj
}

func GetRequests(Id int) (*Requests, error) {
	obj := NewRequests(nil)

	var err error

	if Id != 0 {
		obj.Id = Id
		err = obj.Load()
	}

	return obj, err
}

func MakeDate(dateDay bool, dateInterval time.Duration) (dateFrom, dateTo int, err error) {
	var (
		qry  string
		arg  = make([]interface{}, 0)
		rows *sql.Rows
	)

	if dateDay {
		qry = "SELECT UNIXTIME(DATE_SUB(NOW(), INTERVAL 1 DAY)), UNIXTIME(NOW())"
	} else {
		qry = "SELECT UNIXTIME(DATE_SUB(NOW(), INTERVAL ? SECOND)), UNIXTIME(NOW())"
		arg = []interface{}{dateInterval.Seconds()}
	}

	rows, err = GetDbCli().Query(qry, arg...)

	if err != nil {
		return
	} else if err = rows.Err(); err != nil {
		return
	}

	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&dateFrom, &dateTo); err != nil {
			return
		} else if err = rows.Err(); err != nil {
			return
		}

		DebugLevel.Logf("Make Date for report : %d -> %d (Day Mode: %v, Interval: %s)", dateFrom, dateTo, dateDay, dateInterval.String())
		break
	}

	if err = rows.Err(); err != nil {
		return
	}

	return
}

func (obj *Requests) SetLocked() error {
	res, err := GetDbCli().Exec(fmt.Sprintf("UPDATE `%s`", obj.table)+" SET `locked` = ? WHERE `id`=? LIMIT 1", true, obj.Id)

	if err != nil {
		return err
	}

	if row, err := res.RowsAffected(); err != nil {
		return err
	} else if row != 0 {
		DebugLevel.Logf("Updated %d row into table %s : %s (id: %d)", row, obj.table, obj.Repuri, obj.Id)
	}

	obj.Locked = true
	return nil
}

func (obj *Requests) SetUnLocked() error {
	res, err := GetDbCli().Exec(fmt.Sprintf("UPDATE `%s`", obj.table)+" SET `locked` = ? WHERE `id`=? LIMIT 1", false, obj.Id)

	if err != nil {
		return err
	}

	if row, err := res.RowsAffected(); err != nil {
		return err
	} else if row != 0 {
		DebugLevel.Logf("Updated %d row into table %s : %s (id: %d)", row, obj.table, obj.Repuri, obj.Id)
	}

	obj.Locked = false
	return nil
}

func (obj Requests) IsLocked() bool {
	return obj.Locked
}

func (obj *Requests) Load() error {
	var (
		rows *sql.Rows
		err  error
		qry  = fmt.Sprintf("SELECT %s FROM `%s`", field_requests, obj.table)
		arg  = make([]interface{}, 0)
	)

	if obj.Id != 0 {
		arg = []interface{}{obj.Id}
		qry = qry + " WHERE `id`=? LIMIT 1"
	} else if obj.Domain != nil && obj.Domain.Id != 0 {
		arg = []interface{}{obj.Domain.Id}
		qry = qry + " WHERE `domain`=? LIMIT 1"
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
		var dom int

		err = rows.Scan(
			&obj.Id,
			&obj.Date,
			&dom,
			&obj.Repuri,
			&obj.Pct,
			&obj.Policy,
			&obj.Spolicy,
			&obj.ASPF,
			&obj.ADKIM,
			&obj.Locked,
		)

		if err != nil {
			return err
		} else if err = rows.Err(); err != nil {
			return err
		}

		if dom > 0 {
			obj.Domain, _ = GetDomain(dom)
		} else {
			obj.Domain = NewDomain("")
		}

		DebugLevel.Logf("Find row into table %s : %s (id: %d)", obj.table, obj.Repuri, obj.Id)
		break
	}

	if err = rows.Err(); err != nil {
		return err
	}

	return nil
}

func (obj *Requests) Save() error {
	var (
		res sql.Result
		row int64
		nbr int64
		err error
	)

	if obj.Id != 0 {
		return obj.Update()
	}

	if obj.Repuri == "" {
		return fmt.Errorf("cannot add an empty row into table %s", obj.table)
	}

	if obj.Date.IsZero() {
		obj.Date = time.Now()
	}

	fld := strings.SplitN(field_requests, ",", 2)
	lst := strings.TrimSpace(fld[1])

	res, err = GetDbCli().Exec(
		fmt.Sprintf("INSERT INTO `%s`(%s) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)", obj.table, lst),
		&obj.Date,
		&obj.Domain.Id,
		&obj.Repuri,
		&obj.Pct,
		&obj.Policy,
		&obj.Spolicy,
		&obj.ASPF,
		&obj.ADKIM,
		&obj.Locked,
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
		DebugLevel.Logf("Added %d row into table %s : %s (id: %d)", row, obj.table, obj.Repuri, obj.Id)
	}

	return nil
}

func (obj *Requests) Update() error {
	var (
		res sql.Result
		row int64
		err error
	)

	if obj.Id == 0 {
		return obj.Save()
	}

	if obj.Repuri == "" {
		return fmt.Errorf("cannot update an empty row into table %s", obj.table)
	}

	sql := fmt.Sprintf("UPDATE `%s` SET `repuri` = ?", obj.table)
	arg := []interface{}{obj.Repuri}

	if obj.Domain.Id != 0 {
		sql = sql + ", `domain` = ? "
		arg = append(arg, obj.Domain.Id)
	}

	if obj.Pct != 0 {
		sql = sql + ", `pct` = ? "
		arg = append(arg, obj.Pct)
	}

	if obj.Policy != 0 {
		sql = sql + ", `policy` = ? "
		arg = append(arg, obj.Policy)
	}

	if obj.Spolicy != 0 {
		sql = sql + ", `spolicy` = ? "
		arg = append(arg, obj.Spolicy)
	}

	if obj.ASPF != 0 {
		sql = sql + ", `aspf` = ? "
		arg = append(arg, obj.ASPF)
	}

	if obj.ADKIM != 0 {
		sql = sql + ", `adkim` = ? "
		arg = append(arg, obj.ADKIM)
	}

	if !obj.Date.IsZero() {
		sql = sql + ", `date` = ? "
		arg = append(arg, obj.Date)
	}

	sql = sql + ", `locked` = ? "
	arg = append(arg, obj.Locked, obj.Id)
	res, err = GetDbCli().Exec(sql+" WHERE `id`=? LIMIT 1", arg...)

	if err != nil {
		return err
	}

	row, err = res.RowsAffected()

	if err != nil {
		return err
	}

	if row != 0 {
		DebugLevel.Logf("Updated %d row into table %s : %s (id: %d)", row, obj.table, obj.Repuri, obj.Id)
	}

	return nil
}

func (obj *Requests) Delete() error {
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
		DebugLevel.Logf("Deleted %d row into table %s : %s (id: %d)", row, obj.table, obj.Repuri, obj.Id)
	}

	return nil
}

func (obj *Requests) SetDate(received string) error {
	var (
		val time.Time
		nix int64
		err error
	)

	if nix, err = strconv.ParseInt(received, 10, 64); err == nil {
		return obj.SetDateTime(time.Unix(nix, 0))
	}

	if val, err = time.Parse("2018-07-14T22:08:13+02:00", received); err != nil {
		return err
	}

	return obj.SetDateTime(val)
}

func (obj *Requests) SetDateTime(received time.Time) error {
	if obj.Date.IsZero() || obj.Date.After(received) {
		obj.Date = received
	}

	return nil
}

func (obj *Requests) GetReport(org, email string, upd, sent bool, dateMode bool, dateInterval time.Duration) (report.Report, error) {
	if obj.IsLocked() {
		return nil, errors.New("cannot generate report for a locked request")
	} else if err := obj.SetLocked(); err != nil {
		return nil, err
	}

	df, de, err := GetRangeDate(obj, sent, dateMode, dateInterval)
	if err != nil {
		return nil, err
	}

	lst, err := GetAllMessages(obj, sent, dateMode, dateInterval)
	if err != nil {
		return nil, err
	}

	var msg = make([]report.ReportRecord, 0)

	for _, m := range lst {
		if r, e := m.GetReport(); e != nil {
			return nil, err
		} else {
			msg = append(msg, r)
		}
	}

	rep := report.GetReport(
		obj.Repuri,
		report.GetReportMetadata(org, email, fmt.Sprintf("%s-%d", obj.Domain.Name, time.Now().Unix()), df, de),
		report.GetReportPolicy(obj.Domain.Name, obj.GetADKIM(), obj.GetASPF(), obj.GetPolicy(), obj.GetSPolicy(), obj.Pct),
		msg,
	)

	if upd {
		for _, m := range lst {
			m.Sent = true
			err = m.Save()
			ErrorLevel.LogErrorCtx(true, fmt.Sprintf("saving sent messages '%s' (ID: %d)", m.JobId, m.Id), err)
		}
	}

	return rep, obj.SetUnLocked()
}

func (obj Requests) GetADKIM() string {
	switch obj.ADKIM {
	case 114:
		return "r"
	case 115:
		return "s"
	default:
		return "unknown"
	}
}

func (obj Requests) GetASPF() string {
	switch obj.ASPF {
	case 114:
		return "r"
	case 115:
		return "s"
	default:
		return "unknown"
	}
}

func (obj Requests) GetPolicy() string {
	switch obj.Policy {
	case 110:
		return "none"
	case 113:
		return "quarantine"
	case 114:
		return "reject"
	default:
		return "unknown"
	}
}

func (obj Requests) GetSPolicy() string {
	switch obj.Spolicy {
	case 110:
		return "none"
	case 113:
		return "quarantine"
	case 114:
		return "reject"
	default:
		return "unknown"
	}
}
