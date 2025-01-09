# chttp

This library is a tool to more conveniently obtain the required content in the corresponding set from the request of the go-chi library.

## Usage
```go
type BaseReq struct {
TraceId    *string `header:"traceId,omitempty" v:"required"`
Platform   *string `header:"platform,omitempty" default:"whatsapp"`
ReqId      *string `header:"reqId,omitempty" `
}
type TranferStoreReq struct {
	BaseReq         `cv:"true"`
	Origin *string  `url:"origin" v:"required"`
	StoreId  *string `url:"storeId" v:"required"`
	UserId         *string `json:"userId,omitempty"  v:"required"`
	TransferType   *string `json:"transferType,omitempty"  v:"required"`
}
```
## Tags Definitions
### Value
```go
`json:"<field>"` // value fetch from json body
`header:"<field>"` // value fetch from header
`param:"<field>"` // value fetch from url param
`url:"<field>"` // value fetch from url (only support for go-chi lib)
```
### Func
```go
`v:"required"`// to tell chttp to validate this field != nill
`cv:"true"` // to tell chttp to perform recursion with this struct
`default:"<value: string|int|float|bool>"` // to set the default value
```
## Handle
```go
package cthttp

import (
	"github.com/kuah/chttp"
	"net/http"
)
// this is a sample to show you that I just need to know valid or not in my business
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

## Further Usage 
```go
func TransferStore(w http.ResponseWriter, r *http.Request) {
    req, isPassed := cthttp.Valid[vo.TranferStoreReq](r, w)
    if isPassed == false {
    return
    }
    // your code ...
}
```
