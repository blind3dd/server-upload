package logger

import (
	"github.com/sirupsen/logrus"
	"os"
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

var DefaultLogger *logrus.Logger = logrus.New()
