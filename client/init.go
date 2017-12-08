// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package client

import (
	"github.com/farshidtz/elog"
)

var logger *elog.Logger

func init() {
	logger = elog.New("[sc] ", &elog.Config{
		DebugPrefix: "[sc-debug] ",
	})
}
