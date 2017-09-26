// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package main

import (
	"log"
	"os"
	"strconv"

	"code.linksmart.eu/sc/service-catalog/catalog"
)

var logger *log.Logger

func init() {
	logger = log.New(os.Stdout, catalog.LoggerPrefix, 0)

	v, err := strconv.Atoi(os.Getenv("DEBUG"))
	if err == nil && v == 1 {
		logger.SetFlags(log.Ltime | log.Lshortfile)
	}
}
