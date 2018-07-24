package database

import (
	"database/sql"
	"time"

	"fmt"

	"strings"

	"github.com/nabbar/opendmarc-reports/config"
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
type FieldList map[string]string

func (fld FieldList) Join() string {
	var res = make([]string, 0)

	for k, d := range fld {
		if k != "" && d != "" {
			res = append(res, fmt.Sprintf("`%s` %s", k, d))
		}
	}

	return strings.Join(res, ",")
}

type IndexList map[string]map[string]string

func (fld IndexList) Join() string {
	var res = make([]string, 0)

	for k, s := range fld {
		if k == "" {
			continue
		}

		var (
			t, f string
			ok   bool
		)

		if t, ok = s["type"]; !ok {
			continue
		}

		if f, ok = s["fields"]; !ok {
			continue
		}

		t = strings.ToUpper(t)

		var fl = make([]string, 0)
		for _, v := range strings.Split(f, ",") {
			fl = append(fl, fmt.Sprintf("`%s`", v))
		}

		if t == "PRIMARY" {
			res = append(res, fmt.Sprintf("%s KEY (%s)", t, strings.Join(fl, ",")))
		} else {
			res = append(res, fmt.Sprintf("%s KEY `%s` (%s)", t, k, strings.Join(fl, ",")))
		}
	}

	return strings.Join(res, ",")
}

type Generic struct {
	table    string
	fctField func() FieldList
	fctIndex func() IndexList

	Id   int
	Name string
	Date time.Time
}

var (
	useUTC = false
	dbcli  *sql.DB
)

func GetDbCli() *sql.DB {
	if dbcli == nil {
		dbcli = config.GetConfig().GetDatabase()
		InfoLevel.Logf("Database connection is opened")
	}

	return dbcli
}

func Close() {
	if dbcli != nil {
		err := dbcli.Close()
		FatalLevel.LogErrorCtx(InfoLevel, "closing mysql database connection", err)
		dbcli = nil
	}
}

func CheckTables() {
	if err := NewDomain("").CheckTable(); err != nil {
		FatalLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("checking table '%s' exists", table_domains), err)
	}

	if err := NewIpAddr("").CheckTable(); err != nil {
		FatalLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("checking table '%s' exists", table_ipaddr), err)
	}

	if err := NewReporters("").CheckTable(); err != nil {
		FatalLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("checking table '%s' exists", table_reporters), err)
	}

	if err := NewMessages("").CheckTable(); err != nil {
		FatalLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("checking table '%s' exists", table_messages), err)
	}

	if err := NewSignatures(nil).CheckTable(); err != nil {
		FatalLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("checking table '%s' exists", table_signatures), err)
	}

	if err := NewRequests(nil).CheckTable(); err != nil {
		FatalLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("checking table '%s' exists", table_requests), err)
	}
}

func (gen Generic) CheckTable() error {
	if rows, err := GetDbCli().Query(fmt.Sprintf("SHOW TABLES LIKE '%s'", gen.table)); err != nil {
		return err
	} else if err = rows.Err(); err != nil {
		return err
	} else {
		defer rows.Close()
		for rows.Next() {
			var tbl string

			if err = rows.Scan(&tbl); err != nil {
				return err
			} else if err = rows.Err(); err != nil {
				return err
			}

			if tbl == gen.table {
				return nil
			}
		}
	}

	var (
		res sql.Result
		row int64
		err error
	)

	qry := fmt.Sprintf("CREATE TABLE `%s`(%s,%s) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci", gen.table, gen.fctField().Join(), gen.fctIndex().Join())

	if res, err = GetDbCli().Exec(qry); err != nil {
		return err
	} else if row, err = res.RowsAffected(); err != nil {
		return err
	} else if row != 0 {
		DebugLevel.Logf("Table %s : Created (%d affected rows)", gen.table, row)
	}

	return nil
}

func (gen *Generic) Load() error {
	var (
		rows *sql.Rows
		err  error
	)

	if gen.Id != 0 {
		rows, err = GetDbCli().Query(fmt.Sprintf("SELECT `id`, `name`, `date` FROM `%s` WHERE `id`=? LIMIT 1", gen.table), gen.Id)
	} else if gen.Name != "" {
		rows, err = GetDbCli().Query(fmt.Sprintf("SELECT `id`, `name`, `date` FROM `%s` WHERE `name`=? LIMIT 1", gen.table), gen.Name)
	} else {
		return fmt.Errorf("cannot load null row into table %s", gen.table)
	}

	if err != nil {
		return err
	} else if err = rows.Err(); err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&gen.Id, &gen.Name, &gen.Date); err != nil {
			return err
		} else if err = rows.Err(); err != nil {
			return err
		}

		break
	}

	if err = rows.Err(); err != nil {
		return err
	}

	if gen.Id == 0 {
		gen.Save()
	}

	return nil
}

func (gen *Generic) Save() error {
	var (
		res sql.Result
		row int64
		nbr int64
		err error
	)

	if gen.Id != 0 {
		return gen.Update()
	}

	if gen.Name == "" {
		return fmt.Errorf("cannot add an empty row into table %s", gen.table)
	}

	if gen.Date.IsZero() {
		gen.Date = time.Now()
	}

	res, err = GetDbCli().Exec(fmt.Sprintf("INSERT INTO `%s`(`name`, `date`) VALUES(?, ?)", gen.table), gen.Name, gen.Date)

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

		gen.Id = int(nbr)
		DebugLevel.Logf("Added %d row into table %s : %s (id: %d)", row, gen.table, gen.Name, gen.Id)
	}

	return nil
}

func (gen *Generic) Update() error {
	var (
		res sql.Result
		row int64
		err error
	)

	if gen.Id == 0 {
		return gen.Save()
	}

	if gen.Name == "" {
		return fmt.Errorf("cannot update an empty row into table %s", gen.table)
	}

	if gen.Date.IsZero() {
		gen.Date = time.Now()
	}

	res, err = GetDbCli().Exec(fmt.Sprintf("UPDATE `%s` SET `name`=?, `date`=? WHERE `id`=? LIMIT 1", gen.table), gen.Name, gen.Date, gen.Id)

	if err != nil {
		return err
	}

	row, err = res.RowsAffected()

	if err != nil {
		return err
	}

	if row != 0 {
		DebugLevel.Logf("Updated %d row into table %s : %s (id: %d)", row, gen.table, gen.Name, gen.Id)
	}

	return nil
}

func (gen *Generic) Delete() error {
	var (
		res sql.Result
		row int64
		err error
	)

	if gen.Id == 0 {
		return fmt.Errorf("cannot delete an empty or not saved row into table %s", gen.table)
	}

	res, err = GetDbCli().Exec(fmt.Sprintf("DELETE FROM `%s` WHERE `id`=? LIMIT 1", gen.table), gen.Id)

	if err != nil {
		return err
	}

	row, err = res.RowsAffected()

	if err != nil {
		return err
	}

	if row != 0 {
		DebugLevel.Logf("Deleted %d row into table %s : %s (id: %d)", row, gen.table, gen.Name, gen.Id)
	}

	return nil
}
