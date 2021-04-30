package config

import (
	kitlog "github.com/go-kit/kit/log"
	"log"
	"os"
)
var KitLogger  kitlog.Logger
var Logger log.Logger
func init(){
	KitLogger = kitlog.NewLogfmtLogger(os.Stderr)
	KitLogger = kitlog.With(KitLogger, "ts", kitlog.DefaultTimestampUTC)
	KitLogger = kitlog.With(KitLogger, "caller", kitlog.DefaultCaller)
}