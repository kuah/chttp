# chttp

## Definition


## Handle
```go
// this is a sample to show you I just need to know valid or not in my business
func Valid[T any](r *http.Request, w http.ResponseWriter) (T, bool) {
	req, validation, err := chttp.Valid[T](r)
	switch validation {
	case chttp.ParserResultError, chttp.ParserResultNotVerified:
		// you can log error here || use w to response
		return req, false
	case chttp.ParserResultSuccess:
		return req, true
	default:
		return req, false
	}
}

func ReadRequestBody[T any](r *http.Request) (*T, error) {
	return chttp.ReadRequestBody[T](r)
}

```
