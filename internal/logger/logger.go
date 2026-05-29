package logger

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

var (
	logFile *os.File
)

func init() {
	if runtime.GOOS == "windows" {
		enableWindowsANSI()
	}

	var err error
	logFile, err = os.OpenFile("logs.txt", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
	}
}

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

type Level string

const (
	SUCCESS Level = "SUCCESS"
	ERROR   Level = "ERROR"
	WARN    Level = "WARN"
	INFO    Level = "INFO"
	DEBUG   Level = "DEBUG"
)

func Log(level Level, message string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formattedMessage := fmt.Sprintf(message, args...)

	var levelColor string
	switch level {
	case SUCCESS:
		levelColor = colorGreen
	case ERROR:
		levelColor = colorRed
	case WARN:
		levelColor = colorYellow
	case INFO:
		levelColor = colorBlue
	case DEBUG:
		levelColor = colorCyan
	default:
		levelColor = colorReset
	}

	fmt.Printf("%s[%s]%s - %s[%s]%s : %s\n",
		colorGray, timestamp, colorReset,
		levelColor, level, colorReset,
		formattedMessage)

	if logFile != nil {
		fileMsg := fmt.Sprintf("[%s] - [%s] : %s\n", timestamp, level, formattedMessage)
		logFile.WriteString(fileMsg)
	}
}

func Success(message string, args ...interface{}) {
	Log(SUCCESS, message, args...)
}

func Error(message string, args ...interface{}) {
	Log(ERROR, message, args...)
}

func Warn(message string, args ...interface{}) {
	Log(WARN, message, args...)
}

func Info(message string, args ...interface{}) {
	Log(INFO, message, args...)
}

func Debug(message string, args ...interface{}) {
	Log(DEBUG, message, args...)
}
