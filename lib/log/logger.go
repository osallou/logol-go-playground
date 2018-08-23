package logol

import (
    "os"
    "github.com/juju/loggo"
)

func GetLogger(packageName string) (logger loggo.Logger){
    logger = loggo.GetLogger(packageName)
    osDebug := os.Getenv("LOGOL_DEBUG")
    if osDebug != "" && osDebug != "0" {
        logger.SetLogLevel(loggo.DEBUG)
    } else {
        logger.SetLogLevel(loggo.INFO)
    }
    return logger
}
