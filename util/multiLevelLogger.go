package util

import (
    "github.com/Unknwon/log"
)

type Level int


const (
    DEBUG Level= iota
    INFO
    WARNING
    ERROR
    FATAL
)

var LogLevel = INFO


func DebugLog(thing string, args ...interface{}) {
    if LogLevel == DEBUG {
        if len(args) > 0 {
            log.Debug(thing, args)
        } else {
            log.Debug(thing)
        }
    }
}

func InfoLog(thing string, args ...interface{}) {
    if LogLevel >= INFO {
        if len(args) > 0 {
            log.Info(thing, args)
        } else {
            log.Info(thing)
        }
    }
}

func WarningLog(thing string, args ...interface{}) {
    if LogLevel >= WARNING {
        if len(args) > 0 {
            log.Warn(thing, args)
        } else {
            log.Warn(thing)
        }
    }
}

func ErrorLog(thing string, args ...interface{}) {
    if LogLevel >= ERROR {
        if len(args) > 0 {
            log.Error(thing, args)
        } else {
            log.Error(thing)
        }    }
}

func FatalLog(thing string, args ...interface{}) {
    if LogLevel >= FATAL {
        if len(args) > 0 {
            log.Fatal(thing, args)
        } else {
            log.Fatal(thing)
        }
    }
}
