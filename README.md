# rotel-otel-wrapper

This utility wraps the execution of [Rotel](https://github.com/streamfold/rotel)
as if it were invoked like
the OpenTelemetry Collector. It looks for a single `--config <file>` argument
and opens the config file, translating OTEL config into Rotel command
line arguments.

**NOTE**: This has been primarily built for the use of running the OTEL
testbed against Rotel at the moment. It only handles a small number of
scenarios present in the test suite and is not exhaustive. We may expand
it to be more general purpose in the future.

## Install

```shell
go install github.com/streamfold/rotel-otel-wrapper/cmd/rotel-otel-wrapper@latest
```

## Usage

Requires:
* `ROTEL_PATH`: Set to the full path of the Rotel executable

```shell
rotel-otel-wrapper --config /path/to/otel/config.yaml
```

## Developing

```shell
make build && ./dist/rotel-otel-wrapper <args>
```
