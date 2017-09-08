# journal [![godoc](https://img.shields.io/badge/go-documentation-blue.svg)](https://godoc.org/github.com/vaitekunas/journal) [![Go Report Card](https://goreportcard.com/badge/github.com/vaitekunas/journal)](https://goreportcard.com/report/github.com/vaitekunas/journal) [![Build Status](https://travis-ci.org/vaitekunas/journal.svg?branch=master)](https://travis-ci.org/vaitekunas/journal) [![Coverage Status](https://coveralls.io/repos/github/vaitekunas/journal/badge.svg?branch=master)](https://coveralls.io/github/vaitekunas/journal?branch=master)

Package `journal` is intended to be a very simple, but somewhat flexible logging
facility for golang services.

Main attributes:

* Selection of relevant columns for local logs
* Built-in or custom error messages
* Output to file and optionally stdout
* File rotation (daily, weekly, monthly, annually)
* Old log compression
* Remote logging/mirroring via grpc
* Tab-delimited or JSON-formatted output (output to stdout is always tab-delimited)

## Logging locally

```Go
import log "github.com/vaitekunas/log"

logger := log.New(&log.Config{


})

notify := logger.Logfunc("Caller 1")

notify(0, "This is a simple message")
if err := notify(1, "Something went wrong: %s","Some general error"); err != nil {
  notify(999, "Yup, things really went south: %s (%s)", "Should exit the program now","Nothing to be done here")
}

go func(){
    notify := logger.Logfunc("Coroutine 1")
    notify(0, "Starting coroutine")
    notify(3, "")
}()

<- time.After(500*time.Millisecond)

notify(0, "Exiting")
```

## Starting a remote logging facility and logging remotely

Mirroring local logs remotely is a nice way of aggregating information about
a set of your services/instances in one place.

```shell
logserver start-service \
          --host=127.0.0.1 --port=37746 \
          --folder=$HOME/logs/ --file=myservice --stdout=true \
          --rotation=daily --compress=true \
          --cert=$HOME/logger/cert.pem --key=$HOME/logger/key.pem
          --tokens=$HOME/logger/tokens.db
```

### Authentication token management

In order to accept incoming connections we need to create authentication tokens
for each service/instance of the service we wish to allow to connect to this server:

```shell
logserver add-token myservice myinstance
```

This creates a random 64-character token that the `myinstance` instance of
service `myservice` can use to connect to this server. We can also retrieve the
token for the instance:

```shell
logserver show-token myservice myinstance
```

or all instances of a service:

```shell
logserver show-tokens myservice
```

tokens can also be revoked:

```shell
logserver revoke-token myservice myinstance
```

We can also revoke all tokens and close all incoming connections for a given
service:

```shell
logserver revoke-tokens myservice
```

revoking the token(s) closes any associated sessions and denies future connections.

### Remote logging


```Go
import log "github.com/vaitekunas/log"

logger := log.New(&log.Config{


})

notify := logger.Logfunc("Caller 1")

// Connect to the log server
host := "127.0.0.1"
port := 37746
token := "8018A51319ADA07AF801595E18EABE438FF5C330241661B93C9C32ED36D60C00"

if err:= logger.Connect(host, port, instance, token, 5*time.Second); err != nil {
  notify(1, "Could not connect to log server: %s", err.Error())  
}else{
  notify(0, "Connection to logserver (%s:%d) established.", host, port)
}



```

Local output:
```
DATA
```

Remote output:
```
DATA
```

The logging facility server will write all the incoming messages to the same file
(and optionally stdout). In the logfile all the columns are present, even  though
incoming messages might have selected differing columns.

Remote logging is done asynchronously. The request will timeout according to the
defined timeout duration.


# Build

`log` imports the `logrpc` subpackage, which is generated from a protobuf definition,
so that in order to build `log` you first need to generate the golang stubs for it:

```shell
protoc --go_out=plugins=grpc:. protobuf/log.proto
```

# TODO

 - [ ] Implement `journal.ConnectToKafka`
