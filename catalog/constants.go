// Copyright 2014-2016 Fraunhofer Institute for Applied Information Technology FIT

package catalog

const (
	DNSSDServiceType    = "_linksmart-sc._tcp"
	MaxPerPage          = 100
	ApiVersion          = "1.0.0"
	ApiCollectionType   = "ServiceCatalog"
	ApiRegistrationType = "Service"
	LoggerPrefix        = "[sc] "

	// MetaKeyGCExpose is the meta key indicating
	// that the service needs to be tunneled in GC
	MetaKeyGCExpose = "gc_expose"

	CatalogBackendMemory  = "memory"
	CatalogBackendLevelDB = "leveldb"
	StaticLocation        = "/static"
)
