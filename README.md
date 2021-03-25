# SDA Download
`sda-download` is a `go` implementation of the [Data Out API](https://neic-sda.readthedocs.io/en/latest/dataout.html#rest-api-endpoints).

## Configuration
Configuration is done with environment variables, which can be set manually or loaded from [.env](.env).

## Run
Requires database to be running beforehand.
```
go run cmd/main.go
```
