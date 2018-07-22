package database

import (
	"database/sql"
	"fmt"

	"github.com/nabbar/opendmarc-reports/logger"
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

const table_signatures = "signatures"

type Signatures struct {
	Generic

	Id      int
	Message *Messages
	Domain  *Domain
	Pass    int
	Error   bool
}

func NewSignatures(Message *Messages) *Signatures {
	obj := &Signatures{
		Generic: Generic{
			table: table_signatures,
			fctField: func() FieldList {
				return FieldList{
					"id":      "int(11) NOT NULL AUTO_INCREMENT",
					"message": "int(11) NOT NULL DEFAULT '0'",
					"domain":  "int(11) NOT NULL DEFAULT '0'",
					"pass":    "tinyint(4) NOT NULL DEFAULT '0'",
					"error":   "tinyint(4) NOT NULL DEFAULT '0'",
				}
			},
			fctIndex: func() IndexList {
				return IndexList{
					"PRIMARY": {"type": "PRIMARY", "fields": "id"},
					"message": {"type": "UNIQUE", "fields": "id,message"},
				}
			},
		},
	}

	if Message != nil && Message.Id != 0 {
		obj.Message = Message
	}

	return obj
}

func GetSignatures(Id int) (*Signatures, error) {
	obj := NewSignatures(nil)

	var err error

	if Id != 0 {
		obj.Id = Id
		err = obj.Load()
	}

	return obj, err
}

func GetAllSignatures(Message *Messages) ([]*Signatures, error) {
	var (
		res  = make([]*Signatures, 0)
		rows *sql.Rows
		err  error
	)

	sql := fmt.Sprintf("SELECT `id`,`message`,`domain`,`pass`,`error` FROM `%s`", table_signatures)
	rows, err = GetDbCli().Query(sql+" WHERE `message`=?", Message.Id)

	if err != nil {
		return res, err
	} else if err = rows.Err(); err != nil {
		return res, err
	}

	defer rows.Close()

	for rows.Next() {
		obj := NewSignatures(Message)
		dom := 0
		msg := 0

		if err = rows.Scan(&obj.Id, &msg, &dom, &obj.Pass, &obj.Error); err != nil {
			return res, err
		} else if err = rows.Err(); err != nil {
			return res, err
		}

		if msg > 0 {
			if msg == Message.Id {
				obj.Message = Message
			} else {
				obj.Message, _ = GetMessages(msg)
			}
		}
		if obj.Message == nil {
			obj.Message = NewMessages("")
		}

		if dom > 0 {
			if dom == Message.FromDomain.Id {
				obj.Domain = Message.FromDomain
			} else if dom == Message.EnvDomain.Id {
				obj.Domain = Message.EnvDomain
			} else if dom == Message.PolicyDomain.Id {
				obj.Domain = Message.PolicyDomain
			} else {
				obj.Domain, _ = GetDomain(dom)
			}
		}
		if obj.Domain == nil {
			obj.Domain = NewDomain("")
		}

		logger.DebugLevel.Logf("Find row into table %s : Job ref %s (id: %d)", obj.table, obj.Message.JobId, obj.Id)
		res = append(res, obj)
	}

	if err = rows.Err(); err != nil {
		return res, err
	}

	return res, nil
}

func (obj Signatures) GetMessage() *Messages {
	return obj.Message
}

func (obj *Signatures) Load() error {
	var (
		rows *sql.Rows
		err  error
		dom  int
		msg  int
	)

	sql := fmt.Sprintf("SELECT `id`,`message`,`domain`,`pass`,`error` FROM `%s`", obj.table)

	if obj.Id != 0 {
		rows, err = GetDbCli().Query(sql+" WHERE `id`=? LIMIT 1", obj.Id)
	} else if obj.Message.Id != 0 {
		rows, err = GetDbCli().Query(sql+" WHERE `message`=? LIMIT 1", obj.Message.Id)
	} else {
		return fmt.Errorf("cannot load null row into table %s", obj.table)
	}

	if err != nil {
		return err
	} else if err = rows.Err(); err != nil {
		return err
	}

	defer rows.Close()

	for rows.Next() {
		if err = rows.Scan(&obj.Id, &msg, &dom, &obj.Pass, &obj.Error); err != nil {
			return err
		} else if err = rows.Err(); err != nil {
			return err
		}

		if msg > 0 {
			if obj.Message != nil && msg == obj.Message.Id {
				obj.Message = obj.Message
			} else {
				obj.Message, _ = GetMessages(msg)
			}
		}
		if obj.Message == nil {
			obj.Message = NewMessages("")
		}

		if dom > 0 {
			if dom == obj.Message.FromDomain.Id {
				obj.Domain = obj.Message.FromDomain
			} else if dom == obj.Message.EnvDomain.Id {
				obj.Domain = obj.Message.EnvDomain
			} else if dom == obj.Message.PolicyDomain.Id {
				obj.Domain = obj.Message.PolicyDomain
			} else {
				obj.Domain, _ = GetDomain(dom)
			}
		}
		if obj.Domain == nil {
			obj.Domain = NewDomain("")
		}

		logger.DebugLevel.Logf("Find row into table %s : %s (id: %d)", obj.table, obj.Message, obj.Id)
		break
	}

	if err = rows.Err(); err != nil {
		return err
	}

	return nil
}

func (obj *Signatures) Save() error {
	var (
		res sql.Result
		row int64
		nbr int64
		err error
	)

	obj.Domain.Save()

	if obj.Id != 0 {
		return obj.Update()
	}

	if obj.Message.Id == 0 {
		return fmt.Errorf("cannot add an empty row into table %s", obj.table)
	}

	res, err = GetDbCli().Exec(fmt.Sprintf("INSERT INTO `%s`(`message`,`domain`,`pass`,`error`) VALUES(?, ?, ?, ?)", obj.table), obj.Message.Id, obj.Domain.Id, obj.Pass, obj.Error)

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
		logger.DebugLevel.Logf("Added %d row into table %s : %d (id: %d)", row, obj.table, obj.Message, obj.Id)
	}

	return nil
}

func (obj *Signatures) Update() error {
	var (
		res sql.Result
		row int64
		err error
	)

	if obj.Id == 0 {
		return obj.Save()
	}

	if obj.Message.Id == 0 {
		return fmt.Errorf("cannot update an empty row into table %s", obj.table)
	}

	sql := fmt.Sprintf("UPDATE `%s` SET `message` = ?", obj.table)
	arg := []interface{}{obj.Message.Id}

	if obj.Domain.Id != 0 {
		sql = sql + ", `domain` = ? "
		arg = append(arg, obj.Domain)
	}

	if obj.Pass != 0 {
		sql = sql + ", `pass` = ? "
		arg = append(arg, obj.Pass)
	}

	sql = sql + ", `error` = ? "
	arg = append(arg, obj.Error, obj.Id)
	res, err = GetDbCli().Exec(sql+" WHERE `id`=? LIMIT 1", arg...)

	if err != nil {
		return err
	}

	row, err = res.RowsAffected()

	if err != nil {
		return err
	}

	if row != 0 {
		logger.DebugLevel.Logf("Updated %d row into table %s : %d (id: %d)", row, obj.table, obj.Message, obj.Id)
	}

	return nil
}

func (obj *Signatures) Delete() error {
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
		logger.DebugLevel.Logf("Deleted %d row into table %s : %d (id: %d)", row, obj.table, obj.Message, obj.Id)
	}

	return nil
}

func (obj *Signatures) DeleteFromMessage() error {
	var (
		res sql.Result
		row int64
		err error
	)

	if obj.Message.Id == 0 {
		return fmt.Errorf("cannot delete all rows in table %s where message id is empty", obj.table)
	}

	res, err = GetDbCli().Exec(fmt.Sprintf("DELETE FROM `%s` WHERE `message`=?", obj.table), obj.Message.Id)

	if err != nil {
		return err
	}

	row, err = res.RowsAffected()

	if err != nil {
		return err
	}

	if row != 0 {
		logger.DebugLevel.Logf("Deleted %d row into table %s : Job %s (Msg Id: %d)", row, obj.table, obj.Message.JobId, obj.Message.Id)
	}

	return nil
}

func (obj Signatures) GetReport() report.ReportDKIM {
	return report.GetReportDKIM(obj.Domain.Name, obj.GetPass())
}

func (obj Signatures) GetPass() string {
	switch obj.Pass {
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
