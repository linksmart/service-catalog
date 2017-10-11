// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

import (
	"github.com/farshidtz/elog"
)

var logger *elog.Logger

func init() {
	logger = elog.New(LoggerPrefix, &elog.Config{
		DebugPrefix: LoggerPrefix,
	})
}
