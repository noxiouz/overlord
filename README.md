# Overlord

Just a simple mock for starting HTTP Cocaine workers locally

## Usage

Copy the binary inside a container with your application.
Overlord starts your worker with the same args and workdir as Docker isolation
plugin under Cocaine.

Run the similar command via docker:

```shell
overlord -http ":8080" -locator "cocaine-dev.host.net" -slave "path/to/ur/slave" -startuptimeout "1m"
```

options:
 + `http` - endpoint to listen incoming HTTP request. This port must be exposed by docker. Default: `:8080`
 + `locator` - comma-separated list of locators for the worker. Default `127.0.0.1:10053,[::1]:10053`
 + `slave` - path to the executable inside the container
 + `startuptimeout` - time to wait for incoming connection from worker
 + `version` - show version and exit

## Build

```shell
make
```
