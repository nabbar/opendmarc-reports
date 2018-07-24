package logger

import (
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"runtime"
	"strings"
	"time"

	"bytes"
	"strconv"

	"io"

	"github.com/sirupsen/logrus"
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
var (
	RootPath string = ""
)

type empty struct{}

func init() {
	RootPath = path.Dir(reflect.TypeOf(empty{}).PkgPath())
}

type Format uint8

const (
	TextFormat Format = iota
	JsonFormat
)

func GetFormatListString() []string {
	return []string{
		strings.ToLower(TextFormat.String()),
		strings.ToLower(JsonFormat.String()),
	}
}

func SetFormat(fmt string) {
	switch strings.ToLower(fmt) {
	case strings.ToLower(TextFormat.String()):
		logrus.SetFormatter(&logrus.TextFormatter{})
	case strings.ToLower(JsonFormat.String()):
		logrus.SetFormatter(&logrus.JSONFormatter{})
	}
}

func (f Format) String() string {
	switch f {
	case JsonFormat:
		return "Json"
	default:
		return "Text"
	}
}

// Level type
type Level uint32

const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	NilLevel
)

func GetLevelListString() []string {
	return []string{
		strings.ToLower(PanicLevel.String()),
		strings.ToLower(FatalLevel.String()),
		strings.ToLower(ErrorLevel.String()),
		strings.ToLower(WarnLevel.String()),
		strings.ToLower(InfoLevel.String()),
		strings.ToLower(DebugLevel.String()),
	}
}

type IOWriter struct {
	lvl Level
	prf string
}

func (iow IOWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	err = nil
	iow.lvl.Log(iow.prf + " " + string(p))
	return
}

func GetIOWriter(level Level, msgPrefixPattern string, msgPrefixArgs ...interface{}) io.Writer {
	return &IOWriter{
		lvl: level,
		prf: fmt.Sprintf(msgPrefixPattern, msgPrefixArgs...),
	}
}

func SetLevel(level string) {
	switch strings.ToLower(level) {
	case strings.ToLower(PanicLevel.String()):
		logrus.SetLevel(logrus.PanicLevel)

	case strings.ToLower(FatalLevel.String()):
		logrus.SetLevel(logrus.FatalLevel)

	case strings.ToLower(ErrorLevel.String()):
		logrus.SetLevel(logrus.ErrorLevel)

	case strings.ToLower(WarnLevel.String()):
		logrus.SetLevel(logrus.WarnLevel)

	case strings.ToLower(DebugLevel.String()):
		logrus.SetLevel(logrus.DebugLevel)

	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
	DebugLevel.Logf("Change Log Level to %s", logrus.GetLevel().String())
}

func (level Level) Uint8() uint8 {
	return uint8(level)
}

// Convert the Level to a string. E.g. PanicLevel becomes "panic".
func (level Level) String() string {
	switch level {
	case DebugLevel:
		return "Debug"
	case InfoLevel:
		return "Info"
	case WarnLevel:
		return "Warning"
	case ErrorLevel:
		return "Error"
	case FatalLevel:
		return "Fatal Error"
	case PanicLevel:
		return "Critical Error"
	}

	return "unknown"
}

func (level Level) Logf(format string, args ...interface{}) {
	level.logDetails(fmt.Sprintf(format, args...), nil)
}

func (level Level) Log(message string) {
	level.logDetails(message, nil)
}

func (level Level) LogData(message string, data interface{}) {
	level.logDetails(message, data)
}

func (level Level) LogError(err error) bool {
	if err != nil {
		level.logDetails(err.Error(), nil)
		return true
	}

	return false
}

func (level Level) LogErrorCtx(levelElse Level, context string, err error) bool {
	if err != nil {
		level.logDetails(fmt.Sprintf("%s while %s : %v", level.String(), context, err), nil)
		return true
	} else if levelElse != NilLevel {
		levelElse.logDetails(fmt.Sprintf("OK : %s", context), nil)
	}

	return false
}

func (level Level) LogErrorCtxf(levelElse Level, contextPattern string, err error, args ...interface{}) bool {
	return level.LogErrorCtx(levelElse, fmt.Sprintf(contextPattern, args...), err)
}

func cleanTraceFile(file string) string {
	if strings.Contains(file, RootPath) {
		ft := strings.SplitAfter(file, RootPath)
		return strings.TrimPrefix(ft[1], "/")
	}

	return file
}

func (level Level) logDetails(message string, data interface{}) {

	var (
		dataJson []byte
		dataStr  string
		err      error
		file     string
		line     int
	)

	_, file, line, _ = runtime.Caller(2)
	file = cleanTraceFile(file)

	if path.Base(file) == "logger.go" && path.Dir(file) == "log" {
		_, file, line, _ = runtime.Caller(3)
		file = cleanTraceFile(file)
	}

	dataStr = ""

	if data != nil {
		if dataJson, err = json.Marshal(data); err == nil {
			dataStr = " -- " + string(dataJson)
		}
	}

	if logrus.GetLevel() == logrus.DebugLevel {
		message = fmt.Sprintf("[%d][%s][%s][%s|#%d] - %s%s", GetGID(), level.String(), time.Now().Format(time.RFC3339Nano), file, line, message, dataStr)
	} else {
		message = fmt.Sprintf("%s : %s%s", level.String(), message, dataStr)
	}

	switch level {
	case DebugLevel:
		logrus.Debugln(message)

	case InfoLevel:
		logrus.Infoln(message)

	case WarnLevel:
		logrus.Warnln(message)

	case ErrorLevel:
		logrus.Errorln(message)

	case FatalLevel:
		logrus.Fatalln(message)

	case PanicLevel:
		logrus.Panicln(message)
	}
}

func (level Level) WithFields(message string, fields map[string]interface{}) {
	logEntry := logrus.WithFields(logrus.Fields(fields))

	switch level {
	case DebugLevel:
		logEntry.Debug(message)

	case InfoLevel:
		logEntry.Info(message)

	case WarnLevel:
		logEntry.Warn(message)

	case ErrorLevel:
		logEntry.Error(message)

	case FatalLevel:
		logEntry.Fatal(message)

	case PanicLevel:
		logEntry.Panic(message)
	}
}

func GetGID() uint64 {
	b := make([]byte, 64)

	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]

	n, _ := strconv.ParseUint(string(b), 10, 64)

	return n
}
