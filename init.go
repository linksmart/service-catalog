// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"code.linksmart.eu/sc/service-catalog/catalog"
	"github.com/farshidtz/elog"
)

var logger *elog.Logger

func init() {
	logger = elog.New(catalog.LoggerPrefix, &elog.Config{
		DebugPrefix: catalog.LoggerPrefix,
		DebugTrace:  elog.NoTrace,
	})
}
