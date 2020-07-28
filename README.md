# argus
(pronounced "ar-gus")

[![Build Status](https://travis-ci.com/xmidt-org/argus.svg?branch=main)](https://travis-ci.com/xmidt-org/argus)
[![codecov.io](http://codecov.io/github/xmidt-org/argus/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/argus?branch=main)
[![Code Climate](https://codeclimate.com/github/xmidt-org/argus/badges/gpa.svg)](https://codeclimate.com/github/xmidt-org/argus)
[![Issue Count](https://codeclimate.com/github/xmidt-org/argus/badges/issue_count.svg)](https://codeclimate.com/github/xmidt-org/argus)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/argus)](https://goreportcard.com/report/github.com/xmidt-org/argus)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/argus/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/release/xmidt-org/argus.svg)](CHANGELOG.md)

## Summary
The [XMiDT](https://xmidt.io/) server for storing webhooks to be used by caduceus. This service is used to replace SNS.
Refer the [overview docs](https://xmidt.io/docs/introduction/overview/)for more information on how argus fits into the overall picture.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Details](#details)
- [Usage](#usage)
- [Build](#build)
- [Deploy](#deploy)
- [Contributing](#contributing)

## Code of Conduct

This project and everyone participating in it are governed by the [XMiDT Code Of Conduct](https://xmidt.io/code_of_conduct/). 
By participating, you agree to this Code.

## Details
argus has one function: interact with a database whether it is internal or external.
To enable this, argus has two endpoints: 1) individual items, and 2) buckets containing items.

### Create Individual Item - `store/{bucket}` endpoint
This endpoint allows for clients to `PUT` an object into argus. The body must be in json format with _identifier_ and
_data_. _ttl_ is an optional field in seconds, if not set it will become 5 minutes. An optional header `X-Midt-Owner` can be sent
to associate the object with an owner. 

An example put object
```json
{
  "identifier" : "earth",
  "data": {
    "year":  1967,
     "words": ["What", "a", "Wonderful", "World"]
  },
  "ttl" : 300
}
```

An example response: 
```json
{
  "bucket" : "world",
  "id":  "ZWFydGjjsMRCmPwcFJr79MiZb7kkJ65B5GSbk0yklZkbeFK4VQ"
}
```

### Bucket - `store/{bucket}` endpoint
This endpoint allows for `GET` to retrieve all the items in the bucket organized by the id.
An example response will look like where "earth" is the id of the item. An optional header `X-Midt-Owner` can be sent
with the request, if supplied the results will be filtered down. If `X-Midt-Owner` is empty then all the items will be returned.

An example response:
```json
{
    "ZWFydGjjsMRCmPwcFJr79MiZb7kkJ65B5GSbk0yklZkbeFK4VQ": {
        "identifier": "earth",
        "data": {
            "words": [
                "What",
                "a",
                "Wonderful",
                "World"
            ],
            "year": 1967
        },
        "ttl": 255
    }
}
```


### Individual Item - `store/{bucket}/{id}` endpoint
This endpoint allows for `GET`, and `DELETE` REST methods to interact with any json object that was created with the previous
`PUT` request. An optional header `X-Midt-Owner` can be sent with the request, if the owner of the object matches the header value
the object is returned, otherwise 404 will be returned. 

An example response:
```json
{
    "identifier": "earth",
    "data": {
        "words": [
            "What",
            "a",
            "Wonderful",
            "World"
        ],
        "year": 1967
    },
    "ttl": 295
}
```


## Build

### Source

In order to build from the source, you need a working Go environment with
version 1.11 or greater. Find more information on the [Go website](https://golang.org/doc/install).

You can directly use `go get` to put the argus binary into your `GOPATH`:
```bash
go get github.com/xmidt-org/argus
```

You can also clone the repository yourself and build using make:

```bash
mkdir -p $GOPATH/src/github.com/xmidt-org
cd $GOPATH/src/github.com/xmidt-org
git clone git@github.com:xmidt-org/argus.git
cd argus
make build
```

### Makefile

The Makefile has the following options you may find helpful:
* `make build`: builds the argus binary
* `make docker`: builds a docker image for argus, making sure to get all
   dependencies
* `make local-docker`: builds a docker image for argus with the assumption
   that the dependencies can be found already
* `make test`: runs unit tests with coverage for argus
* `make clean`: deletes previously-built binaries and object files

### RPM

First have a local clone of the source and go into the root directory of the 
repository.  Then use rpkg to build the rpm:
```bash
rpkg srpm --spec <repo location>/<spec file location in repo>
rpkg -C <repo location>/.config/rpkg.conf sources --outdir <repo location>'
```

### Docker

The docker image can be built either with the Makefile or by running a docker
command.  Either option requires first getting the source code.

See [Makefile](#Makefile) on specifics of how to build the image that way.

For running a command, either you can run `docker build` after getting all
dependencies, or make the command fetch the dependencies.  If you don't want to
get the dependencies, run the following command:
```bash
docker build -t argus:local -f deploy/Dockerfile .
```
If you want to get the dependencies then build, run the following commands:
```bash
go mod vendor
docker build -t argus:local -f deploy/Dockerfile.local .
```

For either command, if you want the tag to be a version instead of `local`,
then replace `local` in the `docker build` command.

### Kubernetes

A helm chart can be used to deploy argus to kubernetes
```
helm install xmidt-argus deploy/helm/argus
```

## Deploy

For deploying a XMiDT cluster refer to [getting started](https://xmidt.io/docs/operating/getting_started/).

For running locally, ensure you have the binary [built](#Source).  If it's in
your `GOPATH`, run:
```
argus
```
If the binary is in your current folder, run:
```
./argus
```

## Contributing

Refer to [CONTRIBUTING.md](CONTRIBUTING.md).
