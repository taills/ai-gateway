package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

var std = log.New(os.Stdout, "", 0)

func format(level, msg string) string {
	return fmt.Sprintf("[%s] [%s] %s", time.Now().Format("2006-01-02 15:04:05"), level, msg)
}

func SysLog(msg string) {
	std.Println(format("INFO", msg))
}

func SysLogf(msg string, args ...any) {
	SysLog(fmt.Sprintf(msg, args...))
}

func Info(ctx context.Context, msg string) {
	SysLog(msg)
}

func Infof(ctx context.Context, msg string, args ...any) {
	SysLogf(msg, args...)
}

func Warn(ctx context.Context, msg string) {
	std.Println(format("WARN", msg))
}

func Warnf(ctx context.Context, msg string, args ...any) {
	Warn(ctx, fmt.Sprintf(msg, args...))
}

func Error(ctx context.Context, msg string) {
	std.Println(format("ERROR", msg))
}

func Errorf(ctx context.Context, msg string, args ...any) {
	Error(ctx, fmt.Sprintf(msg, args...))
}

func FatalLog(msg string) {
	std.Fatal(format("FATAL", msg))
}

func SetupLogger() {
	// logger is already configured; no-op for now
}
