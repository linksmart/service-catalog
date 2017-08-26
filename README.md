# LinkSmart Service Catalog

## Compile from source
```
git clone https://code.linksmart.eu/scm/sc/service-catalog.git src/code.linksmart.eu/sc/service-catalog
export GOPATH=`pwd`
go install code.linksmart.eu/sc/service-catalog
```

## Docker
The following command runs service catalog with the default configurations:
```
docker run -p 8082:8082 docker.linksmart.eu/sc/service-catalog
```
