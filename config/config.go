package config

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"time"

	"errors"

	"strings"

	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pelletier/go-toml"
	"github.com/spf13/viper"
	. "github.com/nabbar/opendmarc-reports/logger"
	"github.com/nabbar/opendmarc-reports/tools"
	"gopkg.in/yaml.v2"
)

/*
Copyright 2017 Nicolas JUHEL

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

const (
	DEFAULT_DATABASE_HOST = "localhost"
	DEFAULT_DATABASE_PORT = 3306
	DEFAULT_DATABASE_NAME = "opendmarc"
	DEFAULT_DATABASE_USER = "opendmarc"
	DEFAULT_DATABASE_PASS = "opendmarc"

	DEFAULT_SMTP_HOST = "localhost"
	DEFAULT_SMTP_PORT = 25
	DEFAULT_SMTP_USER = "postmaster@localdomain"
	DEFAULT_SMTP_PASS = "opendmarc"

	DEFAULT_INTERVAL = "24h"

	DEFAULT_DAT_PATH = "/var/tmp/"
)

func GetDefaultDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", DEFAULT_DATABASE_USER, DEFAULT_DATABASE_PASS, DEFAULT_DATABASE_HOST, DEFAULT_DATABASE_PORT, DEFAULT_DATABASE_NAME)
}

func GetDefaultSmtp() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/tls", DEFAULT_SMTP_USER, DEFAULT_SMTP_PASS, DEFAULT_SMTP_HOST, DEFAULT_SMTP_PORT)
}

type configModel struct {
	Verbose  int  `json:"verbose" yaml:"verbose" toml:"verbose"`
	Testing  bool `json:"testing" yaml:"testing" toml:"testing"`
	NoUpdate bool `json:"noUpdate" yaml:"noUpdate" toml:"noUpdate"`
	Day      bool `json:"yesterday" yaml:"yesterday" toml:"yesterday"`

	Interval string `json:"interval" yaml:"interval" toml:"interval"`
	Utc      bool   `json:"utc" yaml:"utc" toml:"utc"`
	MysqlDSN string `json:"database" yaml:"database" toml:"database"`
	SMTPUrl  string `json:"smtp" yaml:"smtp" toml:"smtp"`

	Domain configDomain `json:"domain" yaml:"domain" toml:"domain"`
	Report configReport `json:"report" yaml:"report" toml:"report"`

	SMTP SMTP `json:"-" yaml:"-" toml:"-"`
}

type Config interface {
	JSON() []byte
	YAML() []byte
	TOML() []byte

	Connect()

	GetInterval() time.Duration
	IsDayMode() bool

	IsTesting() bool
	IsUpdate() bool

	GetOrg() string
	GetEmail() *tools.MailAddress
	GetMakeRecipient(to tools.ListMailAddress) tools.ListMailAddress

	GetDatabase() *sql.DB
	GetSMTP() SMTP
	GetHTTP(url string) HTTP
	GetFTP(url string) FTP
}

type configDomain struct {
	Only    []string `json:"only" yaml:"only" toml:"only"`
	Exclude []string `json:"exclude" yaml:"exclude" toml:"exclude"`
}

type configReport struct {
	Email string `json:"email" yaml:"email" toml:"email"`
	Org   string `json:"org" yaml:"org" toml:"org"`
	Copy  string `json:"copy" yaml:"copy" toml:"copy"`
}

var (
	config             *configModel
	httpclient         *http.Client
	err_empty_file     = errors.New("empty file")
	err_invalid_format = errors.New("invalid file format (json, yaml, toml)")
)

func GetConfig() Config {
	if config == nil {
		loadConfig()
	}
	return config
}

func loadConfig() {
	config = &configModel{
		Verbose:  viper.GetInt("verbose"),
		Testing:  viper.GetBool("testing"),
		NoUpdate: viper.GetBool("noUpdate"),
		Day:      viper.GetBool("yesterday"),

		Interval: formatInterval(viper.GetString("interval")),
		Utc:      viper.GetBool("utc"),
		MysqlDSN: viper.GetString("database"),
		SMTPUrl:  viper.GetString("smtp"),

		Domain: configDomain{
			Only:    viper.GetStringSlice("domain.only"),
			Exclude: viper.GetStringSlice("domain.exclude"),
		},

		Report: configReport{
			Email: viper.GetString("report.email"),
			Org:   viper.GetString("report.org"),
			Copy:  viper.GetString("report.copy"),
		},

		SMTP: nil,
	}

	DebugLevel.Logf("Loaded Config: %s", string(config.YAML()))
}

func formatInterval(str string) string {
	interval, err := time.ParseDuration(str)
	FatalLevel.LogErrorCtx(NilLevel, fmt.Sprintf("parsing duration format for '%s'", str), err)

	return fmt.Sprintf("%s", interval.Truncate(time.Second).String())
}

func (cnf *configModel) Connect() {
	db := cnf.GetDatabase()
	defer func() {
		err := db.Close()
		FatalLevel.LogErrorCtx(InfoLevel, "closing mysql database connection", err)
	}()

	err := db.Ping()
	FatalLevel.LogErrorCtx(DebugLevel, "Ping to mysql database", err)

	cnf.GetSMTP().Check()
}

func (cnf configModel) JSON() []byte {
	str, err := json.Marshal(cnf)
	FatalLevel.LogErrorCtx(DebugLevel, "json encoding config", err)

	return []byte(fmt.Sprintf("%s\n", string(str)))
}

func (cnf configModel) YAML() []byte {
	str, err := yaml.Marshal(cnf)
	FatalLevel.LogErrorCtx(DebugLevel, "yaml encoding config", err)

	return []byte(fmt.Sprintf("---\n%s\n", string(str)))
}

func (cnf configModel) TOML() []byte {
	str, err := toml.Marshal(cnf)
	FatalLevel.LogErrorCtx(DebugLevel, "toml encoding config", err)

	return []byte(fmt.Sprintf("%s\n", string(str)))
}

func (cnf configModel) GetInterval() time.Duration {
	interval, err := time.ParseDuration(cnf.Interval)
	FatalLevel.LogErrorCtx(NilLevel, "parsing duration format for interval", err)

	return interval
}

func (cnf configModel) IsDayMode() bool {
	return cnf.Day
}

func (cnf configModel) GetOrg() string {
	return cnf.Report.Org
}

func (cnf configModel) GetEmail() *tools.MailAddress {
	return tools.MailAddressParser(cnf.Report.Email)
}

func (cnf configModel) GetMakeRecipient(to tools.ListMailAddress) tools.ListMailAddress {
	var lst = tools.NewListMailAddress()
	lst.Merge(to)

	for _, m := range strings.Split(cnf.Report.Copy, ",") {
		lst.AddParseEmail(m)
	}

	return lst
}

func (cnf configModel) IsTesting() bool {
	return cnf.Testing
}

func (cnf configModel) IsUpdate() bool {
	return !cnf.NoUpdate
}

func (cnf configModel) GetDatabase() *sql.DB {
	if !strings.Contains(cnf.MysqlDSN, "parseTime=true") {
		if strings.Contains(cnf.MysqlDSN, "?") {
			cnf.MysqlDSN = cnf.MysqlDSN + "&parseTime=true"
		} else {
			cnf.MysqlDSN = cnf.MysqlDSN + "?parseTime=true"
		}
	}

	db, err := sql.Open("mysql", cnf.MysqlDSN)
	FatalLevel.LogErrorCtx(InfoLevel, "Connect to mysql database", err)

	if cnf.Utc {
		var (
			err error
		)

		_, err = db.Exec("SET TIME_ZONE='+00:00'")
		ErrorLevel.LogErrorCtx(InfoLevel, "setting UTC DB connection mode", err)
	}

	return db
}

func (cnf configModel) GetSMTP() SMTP {
	return newSMTPClient(cnf.SMTPUrl)
}

func (cnf configModel) GetHTTP(url string) HTTP {
	return newHTTPClient(url)
}

func (cnf configModel) GetFTP(url string) FTP {
	return newFTPClient(url)
}
