# SDA Download
`sda-download` is a `go` implementation of the [Data Out API](https://neic-sda.readthedocs.io/en/latest/dataout.html#rest-api-endpoints). The [API Reference](API.md) has example requests and responses.

## Configuration
Configuration variables are set in [config.yaml](config.yaml).

## Run
Requires [sda-db](https://github.com/neicnordic/sda-db) to be running beforehand.
```
go run cmd/main.go
```
