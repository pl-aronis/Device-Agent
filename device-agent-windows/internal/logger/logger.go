package logger

import (
	"log"
)

func Info(v ...any) {
	log.Println("[INFO]", v)
}

func Error(v ...any) {
	log.Println("[ERROR]", v)
}

func Warn(v ...any) {
	log.Println("[WARN]", v)
}

func Fatal(v ...any) {
	log.Fatal("[FATAL]", v)
}
