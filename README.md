# Indexer

The Indexer keeps track of packages and their dependencies. It aims to be a fast registry that provides clean interfaces to index, remove and query packages. Messages can be sent to the Indexer over TCP network.

## Tag

* v1.0.0

## Getting Started

The following is the pre-requisites to build the Indexer:

1. [Golang 1.6.2](https://golang.org/dl/)
1. [Docker Engine 1.11.1](https://docs.docker.com/engine/installation/)

**TL;DR** Run `make all` to lint, test, compile, build and run the Indexer as a Docker container. The Indexer server will be listening at `$DOCKER_HOST:8080`. If Docker Machine is used, the default URL is 192.168.99.100:8080.

The project's [Makefile](Makefile) provides targets to test, compile, build and run the Indexer. The `build` and `run` targets require Docker.

```
make lint
```
Invoke `golint` on the project to ensure style consistency.

```
make test
```
Invoke `go test` on the project, providing test cases and coverage information. Test cases defined in `indexer_test.go` are parallelized and race-detection-enabled to help flush out potential race conditions.

```
make compile
```
Invoke `go build` to compile all `.go` files.

```
make build
```
Invoke `docker build` to build the Indexer's Docker image. The image is tagged as `dev/indexer`. Docker Engine must be reachable for this target to work.

```
make run
```
Invoke `docker run` to run an instance of the Indexer's container. The container listens at `$DOCKER_HOST:8080`. If Docker Machine is used, the default URL is 192.168.99.100:8080. Docker Engine must be reachable for this target to work.

```
make coverage
```
Invoke `go test -coverprofile` on the project to generate coverage reports. Two reports (`indexer.cover` and `server.cover`) are generated and viewable from a web browser.

```
make build-server
```
Invoke `go build` to compile and generate the server executable. This is helpful for creating the non-containerized executable.

```
make test-repeat
```
Repeat `go test -race` 15 times to help flush out race conditions.

```
make test-harness-<platform>
```
Run the DO test harness with `concurrency` and `unluckiness` settings of 100. 

```
make all
```
Invoke the `test`, `compile`, `build` and `run` targets. Docker Engine must be reachable for this target to work.

## Design Rationale (1.0.0)

This section provides some design notes to capture implementation decisions for Indexer 1.0.0.

### Indexer Registry 1.0.0

#### API

The [Indexer](indexer.go) is an `interface` type that provides the following 3 APIs:

```
Index(p *Pkg) string
```

Adds `p` to the registry. It returns `OK\n` if the `p` could be indexed or if it was already present. It returns `FAIL\n` if the `p` cannot be indexed because some of its dependencies aren't indexed yet and need to be installed first.

```
Remove(name string) string
```

Removes package `name` from the registry. It returns `OK\n` if the package `name` could be removed from the index. It returns `FAIL\n` if package `name` could not be removed from the index because some other indexed package depends on it. It returns `OK\n` if package `name` wasn't indexed.

```
Query(name string) string
```

Query for package `name` in the registry. It returns `OK\n` if package `name` is indexed. It returns `FAIL\n` if package `name` isn't indexed.

The implementation of these APIs should utilize channels or the Go standard `sync.Mutex` to synchronize multiple concurrent goroutine accesses.

#### Response

The following is the list of string response code returned to the user:

* `OK\n`
* `FAIL\n`
* `ERROR\n`

For future releases, it may be beneficial to encapsulate the code in a struct where each exported field is tagged with the `json` key to enable responses to be marshalled into JSON data.

#### Registry Structure

In version 1.0.0, the decision was made to favor storage performance over durablility. The [`InMemoryIndexer`](indexer.go) provides an in-memory registry implementation of the `Indexer`. The `registry` is the main storage that holds all packages and their dependencies, defined as a `map[string]*Pkg` type. It is a map of "name-to-object". The rationale of choosing a map as the fundamental data structure is to provide fast search, add and remove capabilities based on package names. The `registry` lifespan is limited by the Indexer's lifespan.

The [`Pkg`](pkg.go) struct encapsulates two attributes of a package; namely, the package name and its dependencies. The package dependencies are represented as a slice of strings where only the dependencies names are recorded. For future implementation, it will be beneficial to replace the slice of string with a slice of `* Pkg`s to support transitive dependencies constraints, and detection of cyclic dependencies.

### TCP Server 1.0.0

The `InMemoryIndexer` APIs are served by a [TCP server](cmd/server/tcpserver.go) at port 8080. Currently, this port isn't configurable.

Two things happenend when the server's `Start()` method is invoked:

1. A TCP listener is initialized to accept incoming messages.
1. A goroutine is created to capture the `SIGINT` siginal as well as errors returned by subsequent message-handling goroutines.

The server's `handleConn()` method processes and validates all incoming messages. I/O operations are performed using the Golang standard [`bufio`](https://golang.org/pkg/bufio/) package to enable buffering. The connection with the client remains alive until an `EOF` is received from the client.

To exit the server, press `ctrl+c` to send a `SIGINT` signal to initiate a shutdown, including closing the server's TCP listener and channels.

## LICENSE

Refer [LICENSE](LICENSE) file.
