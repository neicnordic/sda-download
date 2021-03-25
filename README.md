# SDA Download
`sda-download` is a `go` implementation of the [Data Out API](https://neic-sda.readthedocs.io/en/latest/dataout.html#rest-api-endpoints).

## Configuration
Configuration is done with environment variables, which can be set manually or loaded from [.env](.env).
The default location of `.env` is the root directory, and it can be changed with environment variable `DOT_ENV_FILE=`.

## Run
Requires [sda-db](https://github.com/neicnordic/sda-db) to be running beforehand.
```
go run cmd/main.go
```
