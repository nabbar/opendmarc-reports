package config

import (
	"fmt"

	"errors"

	"crypto/tls"
	"net/smtp"

	"strings"

	"bytes"
	"net"
	"net/url"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nabbar/opendmarc-reports/config/certificates"
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

type smtpConfig struct {
	DSN        string
	Host       string
	Port       int
	User       string
	Pass       string
	Net        string
	TLS        TLSMode
	SkipVerify bool
	ServerName string
}

type smtpClient struct {
	con net.Conn
	cli *smtp.Client
	cfg *smtpConfig
}

type SMTP interface {
	Client() *smtp.Client
	Close()
	Check()

	GetSMTPUrl() string
}

type TLSMode uint8

const (
	TLSNONE TLSMode = iota
	STARTTLS
	TLS
)

func ParseTLSMode(str string) TLSMode {
	switch strings.ToLower(str) {
	case TLS.String():
		return TLS
	case STARTTLS.String():
		return STARTTLS
	}

	return TLSNONE
}

func (tlm TLSMode) String() string {
	switch tlm {
	case TLS:
		return "tls"
	case STARTTLS:
		return "starttls"
	default:
		return "none"
	}
}

var cfgsmtp *smtpConfig

// ParseDSN parses the DSN string to a Config
func newSMTPConfig(dsn string) *smtpConfig {
	var (
		smtpcnf = &smtpConfig{
			DSN: dsn,
		}
	)

	// [user[:password]@][net[(addr)]]/dbname[?param1=value1&paramN=valueN]
	// Find the last '/' (since the password or the net addr might contain a '/')
	if !strings.ContainsRune(dsn, '?') && !strings.ContainsRune(dsn, '/') {
		dsn += "/"
	} else if strings.ContainsRune(dsn, '?') && !strings.ContainsRune(dsn, '/') {
		v := strings.Split(dsn, "?")
		v[len(v)-2] += "/"
		dsn = strings.Join(v, "?")
	}

	foundSlash := false
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '/' {
			foundSlash = true
			var j, k int

			// left part is empty if i <= 0
			if i > 0 {
				// [username[:password]@][protocol[(address)]]
				// Find the last '@' in dsn[:i]
				for j = i; j >= 0; j-- {
					if dsn[j] == '@' {
						// username[:password]
						// Find the first ':' in dsn[:j]
						for k = 0; k < j; k++ {
							if dsn[k] == ':' {
								smtpcnf.Pass = dsn[k+1 : j]
								break
							}
						}
						smtpcnf.User = dsn[:k]

						break
					}
				}

				// [protocol[(address)]]
				// Find the first '(' in dsn[j+1:i]
				for k = j + 1; k < i; k++ {
					if dsn[k] == '(' {
						// dsn[i-1] must be == ')' if an address is specified
						if dsn[i-1] != ')' {
							if strings.ContainsRune(dsn[k+1:i], ')') {
								FatalLevel.LogErrorCtx(NilLevel, "parsing SMTP connection string", errors.New("invalid DSN: did you forget to escape a param value?"))
							}
							FatalLevel.LogErrorCtx(NilLevel, "parsing SMTP connection string", errors.New("invalid DSN: network address not terminated (missing closing brace)"))
						}

						if strings.ContainsRune(dsn[k+1:i-1], ':') {
							h, p, e := net.SplitHostPort(dsn[k+1 : i-1])
							DebugLevel.Logf("SMTP Parsing Host found result host = '%s', port = '%s'", h, p)
							if e == nil && p != "" {
								pint, er := strconv.ParseInt(p, 10, 64)
								if er == nil {
									smtpcnf.Port = int(pint)
								}
								smtpcnf.Host = h
							}
						}

						if smtpcnf.Host == "" || smtpcnf.Port == 0 {
							smtpcnf.Host = dsn[k+1 : i-1]
						}
						break
					}
				}
				smtpcnf.Net = dsn[j+1 : k]
			}

			// dbname[?param1=value1&...&paramN=valueN]
			// Find the first '?' in dsn[i+1:]
			for j = i + 1; j < len(dsn); j++ {
				if dsn[j] == '?' {
					if val, err := url.ParseQuery(dsn[j+1:]); err != nil {
						FatalLevel.LogErrorCtx(NilLevel, "checking params", err)
					} else {
						if val.Get("ServerName") != "" {
							smtpcnf.ServerName = val.Get("ServerName")
						}
						if val.Get("SkipVerify") != "" {
							vi, e := strconv.ParseBool(val.Get("SkipVerify"))
							if e == nil {
								smtpcnf.SkipVerify = vi
							}
						}
					}
					break
				}
			}

			smtpcnf.TLS = ParseTLSMode(dsn[i+1 : j])
			break
		}
	}

	if !foundSlash && len(dsn) > 0 {
		FatalLevel.LogErrorCtx(NilLevel, "checking SMTP connection url", errors.New("invalid DSN: missing the slash separating the database name"))
	}

	return smtpcnf
}

// ParseDSN parses the DSN string to a Config
func newSMTPClient(dsn string) SMTP {
	if cfgsmtp == nil {
		cfgsmtp = newSMTPConfig(dsn)
	}

	if cfgsmtp.DSN != dsn {
		return &smtpClient{
			cfg: newSMTPConfig(dsn),
		}
	}

	return &smtpClient{
		cfg: cfgsmtp,
	}
}

func (cnf *smtpClient) Client() *smtp.Client {
	if cnf.cli == nil {
		var (
			err  error
			addr = cnf.cfg.Host
		)

		if cnf.cfg.Net == "" {
			cnf.cfg.Net = "tcp4"
		}

		if cnf.cfg.ServerName == "" && !strings.HasPrefix(strings.ToLower(cnf.cfg.Net), "unix") {
			cnf.cfg.ServerName = cnf.cfg.Host
		} else if strings.HasPrefix(strings.ToLower(cnf.cfg.Net), "unix") && cnf.cfg.TLS != TLSNONE {
			FatalLevel.LogErrorCtx(NilLevel, "checking smtp config", errors.New("cannot use tls or starttls connection with unix socket connection"))
		}

		if cnf.cfg.Port > 0 {
			addr = fmt.Sprintf("%s:%v", cnf.cfg.Host, cnf.cfg.Port)
		}

		if cnf.cfg.TLS == TLS {
			cnf.con, err = tls.Dial(cnf.cfg.Net, addr, certificates.GetTLSConfig(cnf.cfg.ServerName, cnf.cfg.SkipVerify))
			if ErrorLevel.LogErrorCtx(NilLevel, fmt.Sprintf("trying to intialize SMTP '%s' over tls connection to '%s'", cnf.cfg.Net, addr), err) {
				cnf.cfg.TLS = STARTTLS
				err = nil
			}
		}

		if cnf.cfg.TLS != TLS {
			cnf.con, err = net.Dial(cnf.cfg.Net, addr)
			PanicLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("trying to intialize SMTP '%s' connection to '%s'", cnf.cfg.Net, addr), err)
		}

		cnf.cli, err = smtp.NewClient(cnf.con, addr)
		PanicLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("trying to start SMTP client to host '%s'", addr), err)

		if cnf.cfg.TLS == STARTTLS {
			err = cnf.cli.StartTLS(certificates.GetTLSConfig(cnf.cfg.ServerName, cnf.cfg.SkipVerify))
			PanicLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("trying to STARTTLS with SMTP server '%s'", addr), err)
		}

		if cnf.cfg.User != "" || cnf.cfg.Pass != "" {
			err = cnf.cli.Auth(smtp.PlainAuth("", cnf.cfg.User, cnf.cfg.Pass, addr))
			PanicLevel.LogErrorCtx(InfoLevel, fmt.Sprintf("trying to authentificate with user '%s' to SMTP server '%s'", cnf.cfg.User, addr), err)
		}

		cnf.cli.Extension("8BITMIME")
	}

	return cnf.cli
}

func (cnf *smtpClient) Close() {
	if cnf.cli != nil {
		if e := cnf.cli.Quit(); e != nil {
			//ErrorLevel.LogErrorCtx("ending smtp client connection", e)
			cnf.cli.Close()
		}
	}

	if cnf.con != nil {
		cnf.con.Close()
	}
}

func (cnf *smtpClient) Check() {
	defer cnf.Close()
	cnf.Client()
}

func (cnf smtpClient) GetSMTPUrl() string {
	var (
		buf bytes.Buffer
		tmp bytes.Buffer
	)

	// [username[:password]@]
	if len(cnf.cfg.User) > 0 {
		buf.WriteString(cnf.cfg.User)
		if len(cnf.cfg.Pass) > 0 {
			buf.WriteByte(':')
			buf.WriteString(cnf.cfg.Pass)
		}
		buf.WriteByte('@')
	}

	// [username[:password]@]
	if len(cnf.cfg.Host) > 0 {
		tmp.WriteString(cnf.cfg.Host)
		if cnf.cfg.Port > 0 {
			tmp.WriteByte(':')
			tmp.WriteString(fmt.Sprintf("%d", cnf.cfg.Port))
		}
	}

	// [protocol[(address)]]
	if len(cnf.cfg.Net) > 0 {
		buf.WriteString(cnf.cfg.Net)
		if tmp.Len() > 0 {
			buf.WriteByte('(')
			buf.WriteString(tmp.String())
			buf.WriteByte(')')
		}
	}

	// /dbname
	buf.WriteByte('/')
	buf.WriteString(cnf.cfg.TLS.String())

	// [?param1=value1&...&paramN=valueN]
	var val = &url.Values{}

	if cnf.cfg.ServerName != "" {
		val.Add("ServerName", cnf.cfg.ServerName)
	}

	if cnf.cfg.SkipVerify {
		val.Add("SkipVerify", "true")
	}

	params := val.Encode()

	if len(params) > 2 {
		buf.WriteString("?" + params)
	}

	return buf.String()
}
