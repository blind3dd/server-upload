package logger

import (
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"runtime"
	"strings"
)

func NewLogger() *logrus.Entry {
	fields := getExtraFields()
	entry := logrus.WithFields(logrus.Fields{})
	for k, v := range *fields {
		entry = entry.WithFields(logrus.Fields{
			k: v,
		})
	}
	entry.Logger.SetFormatter(&logrus.JSONFormatter{})
	entry.Logger.SetOutput(os.Stdout)
	entry.Println("hello")

	return entry
}

type callInfo struct {
	packageName string
	fileName    string
	funcName    string
	lineNumber  int
}

func getExtraFields() *logrus.Fields {
	pc, file, line, _ := runtime.Caller(2)
	_, fileName := path.Split(file)
	parts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	pl := len(parts)
	packageName := ""
	fN := parts[pl-1]

	if parts[pl-2][0] == '(' {
		fN = parts[pl-2] + "." + fN
		packageName = strings.Join(parts[0:pl-2], ".")
	} else {
		packageName = strings.Join(parts[0:pl-1], ".")
	}
	ci := &callInfo{
		packageName: packageName,
		fileName:    fileName,
		funcName:    fN,
		lineNumber:  line,
	}

	return &logrus.Fields{
		"packageName": ci.packageName,
		"fileName":    ci.fileName,
		"funcName":    ci.funcName,
		"lineNumber":  ci.lineNumber,
	}
}
