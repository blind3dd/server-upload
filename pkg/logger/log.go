package logger

import (
	"github.com/sirupsen/logrus"
	"os"
	"path"
	"runtime"
	"strings"
)

func NewLogger() *logrus.Logger {
	logger := DefaultLogger
	logger.SetOutput(os.Stdout)
	logger.SetFormatter(&logrus.JSONFormatter{})
	//TODO
	//reqID := GetRequestID(ctx)
	//if reqID == "" {
	//	key, err := uuid.NewV7()
	//	if err != nil {
	//		panic(err)
	//	}
	//	uuidString := key.String()
	//	reqID = fmt.Sprintf("%s", uuidString)
	//}

	/*&logEntry{entry}*/
	return logger
}

type callinfo struct {
	packageName string
	fileName    string
	funcName    string
	lineNumber  int
}

func retrieveCallInfo() *callinfo {
	pc, file, line, _ := runtime.Caller(2)
	_, fileName := path.Split(file)
	parts := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	pl := len(parts)
	packageName := ""
	funcName := parts[pl-1]

	if parts[pl-2][0] == '(' {
		funcName = parts[pl-2] + "." + funcName
		packageName = strings.Join(parts[0:pl-2], ".")
	} else {
		packageName = strings.Join(parts[0:pl-1], ".")
	}

	return &callinfo{
		packageName: packageName,
		fileName:    fileName,
		funcName:    funcName,
		lineNumber:  line,
	}
}

var DefaultLogger *logrus.Logger = logrus.New()
