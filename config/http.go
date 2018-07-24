package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

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

type httpClient struct {
	url *url.URL
	cli *http.Client
}

type HTTP interface {
	Check() bool
	Call(zipFile *bytes.Buffer) (bool, *bytes.Buffer)
}

func newHTTPClient(Url string) HTTP {
	pUrl, err := url.Parse(Url)
	PanicLevel.LogErrorCtx(NilLevel, fmt.Sprintf("parsing url '%s'", Url), err)

	return &httpClient{
		url: pUrl,
		cli: getHttpClient(pUrl.Host),
	}
}

func getHttpClient(serverName string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			// Set this value so that the underlying transport round-tripper
			// doesn't try to auto decode the body of objects with
			// content-encoding set to `gzip`.
			//
			// Refer:
			//    https://golang.org/src/net/http/transport.go?h=roundTrip#L1843
			DisableCompression: true,
			TLSClientConfig:    certificates.GetTLSConfig(serverName, false),
		},
	}
}

func (obj *httpClient) Check() bool {
	obj.doRequest(obj.newRequest(http.MethodHead, nil))
	return true
}

func (obj *httpClient) Call(zipFile *bytes.Buffer) (bool, *bytes.Buffer) {
	return obj.checkResponse(
		obj.doRequest(
			obj.newRequest(http.MethodPost, zipFile),
		),
	)
}

func (obj *httpClient) newRequest(method string, body *bytes.Buffer) *http.Request {
	var reader *bytes.Reader

	if body != nil && body.Len() > 0 {
		reader = bytes.NewReader(body.Bytes())
	}

	req, err := http.NewRequest(method, obj.url.String(), reader)
	PanicLevel.LogErrorCtx(NilLevel, fmt.Sprintf("creating '%s' request to '%s'", method, obj.url.Host), err)

	return req
}

func (obj *httpClient) doRequest(req *http.Request) *http.Response {
	res, err := obj.cli.Do(req)
	PanicLevel.LogErrorCtx(NilLevel, fmt.Sprintf("running request '%s:%s'", req.Method, req.URL.Host), err)

	return res
}

func (obj *httpClient) checkResponse(res *http.Response) (bool, *bytes.Buffer) {
	var buf *bytes.Buffer

	if res.Body != nil {
		bdy, err := ioutil.ReadAll(res.Body)
		if err != nil {
			buf.Write(bdy)
		}
	}

	InfoLevel.Logf("Calling '%s:%s' result %s (Body : %d bytes)", res.Request.Method, res.Request.URL.Host, res.Status, buf.Len())

	return strings.HasPrefix(res.Status, "2"), buf
}
