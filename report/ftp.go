package report

import (
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

func (rep reportFile) SendFtp(uri string) {
	defer func() {
		if r := recover(); r != nil {
			InfoLevel.Logf("Recover Panic Value : %v", r)
			return
		}
	}()

	if config.GetConfig().IsTesting() {
		InfoLevel.Logf("Testing mode : don't upload zip file '%s' to ftp '%s'", rep.GetZipName(), uri)
		return
	}

	cli := config.GetConfig().GetFTP(uri)
	defer cli.Close()
	cli.Store(rep.GetFileName(), rep.zipFile)
}
