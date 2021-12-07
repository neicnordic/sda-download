[![CodeQL](https://github.com/neicnordic/sda-download/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/neicnordic/sda-download/actions/workflows/codeql-analysis.yml)
[![Tests](https://github.com/neicnordic/sda-download/actions/workflows/test.yml/badge.svg)](https://github.com/neicnordic/sda-download/actions/workflows/test.yml)
[![Multilinters](https://github.com/neicnordic/sda-download/actions/workflows/report.yml/badge.svg)](https://github.com/neicnordic/sda-download/actions/workflows/report.yml)
[![codecov](https://codecov.io/gh/neicnordic/sda-download/branch/main/graph/badge.svg?token=ZHO4XCDPJO)](https://codecov.io/gh/neicnordic/sda-download)

# SDA Download
`sda-download` is a `go` implementation of the [Data Out API](https://neic-sda.readthedocs.io/en/latest/dataout.html#rest-api-endpoints). The [API Reference](docs/API.md) has example requests and responses.

## Configuration
Configuration variables are set in [config.yaml](config.yaml).

## Run
Requires [sda-db](https://github.com/neicnordic/sda-db) to be running beforehand.
```
go run cmd/main.go
```
