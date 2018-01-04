// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package utils

import (
	"github.com/farshidtz/elog"
)

var logger *elog.Logger

func init() {
	logger = elog.New("[utils] ", &elog.Config{
		DebugPrefix: "[utils-debug] ",
		DebugTrace:  elog.NoTrace,
	})
}
