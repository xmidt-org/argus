# argus
(pronounced "ar-gus")

[![Build Status](https://travis-ci.com/xmidt-org/argus.svg?branch=main)](https://travis-ci.com/xmidt-org/argus)
[![codecov.io](http://codecov.io/github/xmidt-org/argus/coverage.svg?branch=main)](http://codecov.io/github/xmidt-org/argus?branch=main)
[![Code Climate](https://codeclimate.com/github/xmidt-org/argus/badges/gpa.svg)](https://codeclimate.com/github/xmidt-org/argus)
[![Issue Count](https://codeclimate.com/github/xmidt-org/argus/badges/issue_count.svg)](https://codeclimate.com/github/xmidt-org/argus)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmidt-org/argus)](https://goreportcard.com/report/github.com/xmidt-org/argus)
[![Apache V2 License](http://img.shields.io/badge/license-Apache%20V2-blue.svg)](https://github.com/xmidt-org/argus/blob/main/LICENSE)
[![GitHub release](https://img.shields.io/github/release/xmidt-org/argus.svg)](CHANGELOG.md)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=xmidt-org_argus&metric=alert_status)](https://sonarcloud.io/dashboard?id=xmidt-org_argus)

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

### Create Individual Item - `store/{bucket}/{uuid}` endpoint
This endpoint allows for clients to `PUT` an object into argus.  The placeholder variables in the path must contain:

* _bucket_ - The name used to indicate the resource type of which the stored data represents.  A plural form of a noun word should be used for stylistic reasons.
* _uuid_ - The unique ID within the name space of the containing bucket.  It is recommended this value is the resulting value of a SHA256 calculation, using the unique attributes of the object being represented (e.g. `SHA256(<identifier>)`).  This will be used by argus to determine uniqueness of objects being stored or updated.


The body must be in JSON format with the following attributes:

* _uuid_ - Required.  See above.
* _identifier_ - Required.  Common name of the provided resource.  There is no enforcement of uniqueness across resource of this type.
* _data_ - Required.  RAW JSON to be stored.  Opaque to argus.
* _owner_ - Optional.  Free form string to identify the owner of this object.
* _ttl_ - Optional.  Specified in units of seconds.  Defaults to 0 if omitted, which means this object will not auto expire.

An optional header `X-Midt-Owner` can be sent to associate the object with an owner.  The value of this header will be bound to a new record, which would require the same value passed in a `X-Midt-Owner` header for subsequent reads or modifications.  This in effect creates a secret attribute bound to the life of newly created records.

An example PUT request
```
PUT /store/planets/7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7
```
```json
{
  "uuid": "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7",
  "identifier" : "earth",
  "data": {
    "year":  1967,
     "words": ["What", "a", "Wonderful", "World"]
  },
  "ttl" : 300
}
```

Example responses:
```
HTTP/1.1 201 Created
```
The above response would indicate a new object has been created (no existing object with the given UUID was found).

```
HTTP/1.1 200 OK
```
The above response would indicate an existing object has been updated (existing object with the given UUID was found).  Note that a PUT operation on an existing record may also result in "403 Forbidden" error.


### List - `store/{bucket}` endpoint
This endpoint allows for `GET` to retrieve all the items in the bucket organized by the id.

An example response will look like where "earth" is the id of the item. An optional header `X-Midt-Owner` can be sent with the request.  If supplied, only items with secrets matching the supplied value will be returned in the list.

An example response:
```json
[
  {
    "uuid": "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7",
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
]
```


### Individual Item - `store/{bucket}/{uuid}` endpoint
This endpoint allows for `GET`, and `DELETE` REST methods to interact with any object that was created with the previous `PUT` request.  An optional header `X-Midt-Owner` can be sent with the request.  If supplied, the value is validated against the secret stored with this record.  A mismatch will result in a "403 Forbidden" error.

An example response:
```json
{
  "uuid": "7e8c5f378b4addbaebc70897c4478cca06009e3e360208ebd073dbee4b3774e7",
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
  "ttl": 100
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
* `make docker`: fetches all dependencies from source and builds an argus
   docker image
* `make local-docker`: vendors dependencies and builds an argus docker image
   (recommended for local testing)
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

If you'd like to build it without make, follow these instructions based on your use case:

- Local testing
```bash
go mod vendor
docker build -t argus:local -f deploy/Dockerfile .
```
This allows you to test local changes to a dependency. For example, you can build
a argus image with the changes to an upcoming changes to [webpa-common](https://github.com/xmidt-org/webpa-common) by using the [replace](https://golang.org/ref/mod#go) directive in your go.mod file like so:
```
replace github.com/xmidt-org/webpa-common v1.10.2-0.20200604164000-f07406b4eb63 => ../webpa-common
```
**Note:** if you omit `go mod vendor`, your build will fail as the path `../webpa-common` does not exist on the builder container.

- Building a specific version
```bash
git checkout v0.3.6
docker build -t argus:v0.3.6 -f deploy/Dockerfile .
```

**Additional Info:** If you'd like to stand up a XMiDT docker-compose cluster, read [this](https://github.com/xmidt-org/xmidt/blob/master/deploy/docker-compose/README.md).

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
