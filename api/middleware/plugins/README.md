# Middleware plugins
Custom middleware [plugins](https://pkg.go.dev/plugin) can be used to change how authentication and authorization behaves. [token_middleware.go](./token_middleware.go) is the default middleware which uses JWTs at an OIDC provider for fetching user's permissions in GA4GH visa format, and then parsing them into a list of datasets.

## Plugin rules
The plugin:
- must be a `main` package
- can import `sda-download` internal packages, if built inside the project
- must have a `CustomMiddleware` interface with a `GetDatasets` method which returns the list of datasets as `[]string{}`

## Example plugin
```go
package main

type customMiddleware []string

var CustomMiddleware customMiddleware

func (cm customMiddleware) GetDatasets(c *gin.Context) []string {
    // put your custom middleware logic here
    // the return value must be a slice of strings
    var datasets []string
    // for error cases, raise an http exception and `return nil`
    return datasets
}
```

## Building plugin
The plugin can be built with:
```
go build -buildmode=plugin -o your_middleware.so your_middleware.go
```
This is required when running the web app with `go run`. The [Dockerfile](../../../Dockerfile) will automatically build any plugins in the plugins directory.

## Selecting plugin for runtime
The default middleware plugin is `token_middleware.so`, and can be changed with the `plugins.middleware` configuration variable. The variable must be a filepath that points to a prebuilt plugin binary.
```yaml
plugins:
  middleware: ./api/middleware/plugins/token_middleware.so
```
