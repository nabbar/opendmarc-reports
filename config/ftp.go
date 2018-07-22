package config

import (
	"bytes"
	"fmt"
	"net/url"
	"time"

	"os"
	"path"
	"strings"

	"github.com/nabbar/opendmarc-reports/config/certificates"
	. "github.com/nabbar/opendmarc-reports/logger"
	"github.com/secsy/goftp"
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

type ftpClient struct {
	url *url.URL
	cfg goftp.Config
	cli *goftp.Client
}

type FTP interface {
	Connect()
	Close()
	Store(zipName string, zipFile *bytes.Buffer)
}

func newFTPClient(uri string) FTP {
	pUrl, err := url.Parse(uri)
	FatalLevel.LogErrorCtx(true, fmt.Sprintf("parsing url '%s'", uri), err)

	return &ftpClient{
		url: pUrl,
		cfg: getFtpConfig(pUrl),
	}
}

func getFtpConfig(uri *url.URL) goftp.Config {
	usr := uri.User.Username()
	pwd, ok := uri.User.Password()

	if !ok {
		usr, pwd = "", ""
	}

	return goftp.Config{
		User:               usr,
		Password:           pwd,
		ConnectionsPerHost: 10,
		Timeout:            10 * time.Second,
		Logger:             GetIOWriter(DebugLevel, "FTP Host '%s' Debug Message", uri.Host),
		TLSMode:            goftp.TLSImplicit,
		TLSConfig:          certificates.GetTLSConfig(uri.Host, false),
	}
}

func (obj *ftpClient) Connect() {
	var err error
	obj.cli, err = goftp.DialConfig(obj.cfg, obj.url.Host)
	FatalLevel.LogErrorCtxf(true, "connecting to FTP Host '%s'", err, obj.url.Host)
}

func (obj *ftpClient) Close() {
	if obj.cli != nil {
		obj.cli.Close()
		obj.cli = nil
	}
}

func (obj *ftpClient) Store(zipName string, zipFile *bytes.Buffer) {
	if obj.cli == nil {
		obj.Connect()
	}

	if obj.url.Path != "/" {
		if _, err := obj.cli.Stat(obj.url.Path); err != nil {
			_, err := obj.cli.Mkdir(obj.url.Path)
			FatalLevel.LogErrorCtxf(true, "creating FTP path '%s' to Host '%s'", err, obj.url.Path, obj.url.Host)
		}
	}

	dir := strings.Replace(obj.url.Path, "/", string(os.PathSeparator), -1)
	ful := strings.Replace(path.Join(dir, zipName), string(os.PathSeparator), "/", -1)
	err := obj.cli.Store(ful, bytes.NewReader(zipFile.Bytes()))

	FatalLevel.LogErrorCtxf(true, "storing FTP Zip File '%s' to Host '%s'", err, ful, obj.url.Host)
}
