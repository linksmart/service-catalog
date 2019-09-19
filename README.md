# Service Catalog
[![GoDoc](https://godoc.org/github.com/linksmart/service-catalog?status.svg)](https://godoc.org/github.com/linksmart/service-catalog)
[![Docker Pulls](https://img.shields.io/docker/pulls/linksmart/sc.svg)](https://hub.docker.com/r/linksmart/sc/tags)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/linksmart/service-catalog)](https://github.com/linksmart/service-catalog/releases)
[![Build Status](https://travis-ci.com/linksmart/service-catalog.svg?branch=master)](https://travis-ci.com/linksmart/service-catalog)

LinkSmart Service Catalog is a registry enabling discovery of other web services via a RESTful API or through an MQTT broker.
 
## Getting Started
* Read the [Documentation](https://docs.linksmart.eu/display/SC)

## Deployment
### Configuration
The configuration is possible using a JSON file or by setting environment variables. It is described [here](https://docs.linksmart.eu/display/SC/Configuration).

### Docker
The following command runs the latest release of service catalog with the default configurations:
```
docker run -p 8082:8082 linksmart/sc
```
The index of the RESTful API should now be accessible at: http://localhost:8082

To run on other architectures (e.g. `arm32`, `arm64`), clone this repo and build the image locally first by running:
```
docker build -t linksmart/sc .
```

### Binary Distributions
These are available for released versions and for several platforms [here](https://github.com/linksmart/service-catalog/releases).  

Download and run:
```
./service-catalog-<os-arch> --help
```

## Development
The dependencies of this package are managed by [Go Modules](https://blog.golang.org/using-go-modules).

To compile from source:
```
git clone https://github.com/linksmart/service-catalog.git
cd service-catalog
go build -mod=vendor
```

## Contributing
Contributions are welcome. 

Please fork, make your changes, and submit a pull request. For major changes, please open an issue first and discuss it with the other authors.
