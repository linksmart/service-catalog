# Service Catalog
[![GoDoc](https://godoc.org/github.com/linksmart/service-catalog?status.svg)](https://godoc.org/github.com/linksmart/service-catalog)
[![Docker Pulls](https://img.shields.io/docker/pulls/linksmart/sc.svg)](https://hub.docker.com/r/linksmart/sc/tags)
[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/linksmart/service-catalog.svg)](https://github.com/linksmart/service-catalog/tags)
[![Build Status](https://travis-ci.com/linksmart/service-catalog.svg?branch=master)](https://travis-ci.com/linksmart/service-catalog)

LinkSmart Service Catalog is a registry enabling discovery of other web services via a RESTful API or through an MQTT broker.
 
* [Documentation](https://docs.linksmart.eu/display/SC)
* [Issue Tracking](https://boards.linksmart.eu/issues/?jql=project+%3D+LS+AND+component+%3D+%22Service+Catalog%22)

## Run
The following command runs the latest release of service catalog with the default configurations:
```
docker run -p 8082:8082 linksmart/sc
```
Other images and binary distributions can be found from the documentation.

## Development
The dependencies of this package are managed by [Go Modules](https://blog.golang.org/using-go-modules).

To compile from source:
```
git clone https://github.com/linksmart/service-catalog.git
cd service-catalog
go build -mod=vendor
```