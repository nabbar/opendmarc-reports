package certificates

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

import (
	"crypto/tls"
	"crypto/x509"
	"strings"

	. "github.com/nabbar/opendmarc-reports/logger"
)

func CheckCertificates() bool {
	if certificateKey == "" || certificateCrt == "" {
		return false
	}

	if _, err := tls.X509KeyPair([]byte(certificateCrt), []byte(certificateKey)); err != nil {
		return false
	}

	return true
}

func AddCertPair(cert []tls.Certificate, CertKey, CertCrt string) []tls.Certificate {
	if CertKey == "" || CertCrt == "" {
		return cert
	}

	crt, err := tls.LoadX509KeyPair(CertCrt, CertKey)

	if ErrorLevel.LogErrorCtx(NilLevel, "loading pair config certificate", err) {
		return cert
	}

	return append(cert, crt)
}

func AppendCertificates(cert []tls.Certificate) []tls.Certificate {
	var (
		crt tls.Certificate
		err error
	)

	if !CheckCertificates() {
		return cert
	}

	crt, err = tls.X509KeyPair([]byte(certificateCrt), []byte(certificateKey))

	if ErrorLevel.LogErrorCtx(NilLevel, "loading pair included certificate", err) {
		return cert
	}

	return append(cert, crt)
}

func GetClientCA() *x509.CertPool {
	if strings.Replace(strings.Replace(certificateCAClient, "\n", "", -1), " ", "", -1) == "" {
		return nil
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(certificateCAClient))

	return pool
}

func GetRootCA() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(rootInjectCA))

	return pool
}

func GetTLSConfig(serverName string, skipVerify bool) *tls.Config {
	return &tls.Config{
		RootCAs:            GetRootCA(),
		ClientCAs:          GetClientCA(),
		ServerName:         serverName,
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: skipVerify,
	}
}
