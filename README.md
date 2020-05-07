# argus
(pronounced "ar-gus")

[![Build Status](https://travis-ci.com/xmidt-org/argus.svg?branch=master)](https://travis-ci.com/xmidt-org/argus)
[![codecov.io](http://codecov.io/github/xmidt-org/argus/coverage.svg?branch=master)](http://codecov.io/github/xmidt-org/argus?branch=master)
[![Code Climate](https://codeclimate.com/github/xmidt-org/argus/badges/gpa.svg)](https://codeclimate.com/github/xmidt-org/argus)
[![Issue Count](https://codeclimate.com/github/xmidt-org/argus/badges/issue_count.svg)](https://codeclimate.com/github/xmidt-org/argus)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/argus)](https://goreportcard.com/report/github.com/xmidt-org/argus)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/argus/blob/master/LICENSE)
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

#### Individual Item - `store/{bucket}/{id}[?attributes=<tags>]` endpoint
This endpoint allows for `POST`, `GET`, and `DELETE` REST methods to interact with any json object.
When doing a `POST`, **attributes** is an _optional_ query param where the subsequent comma separated strings add tags
to the item for future filtering when retrieving items from a bucket. For example, `?attributes=stage,beta,flavor,blue`
becomes `stage=beta` AND `flavor=blue`.


#### Bucket - `store/{bucket}[?attributes=<tags>]` endpoint
This endpoint allows for `GET` to retrieve all the items in the bucket organized by the id.
An example response will look like where "earth" is the id of the item.

```
{
    "earth": {
        "year": 1967,
        "words": [
            "What",
            "a",
            "Wonderful",
            "World"
        ]
    }
}
```

`atributes` is an _optional_ query parameter that allows consumers to filter results on. For example, if a bucket has
one item with the tags `stage=beta` AND `flavor=blue`, and there is a request with `?attributes=stage,beta` the item is
returned. Where as, if the request is `?attributes=stage,beta,area,water` no items will be returned. If no attributes are
sent with the request, all items in the bucket will be returned. 


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
