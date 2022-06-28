# Authorizations

---

The HTTP Authorization request header can be used to provide credentials that authenticate a user agent with a server, allowing access to a protected resource.
Read [Authorization](https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Authorization) for more.

## Components
### Encoding
`DEFAULT` encoding is builtin `fns`, config is 
```yaml
authorizations:
  encoding:
    expireMinutes: 1440
```
`JWT` encoding is supplied by `fns-contrib`, read [doc](https://github.com/aacfactory/fns-contrib/tree/main/authorizations/encoding/jwt) for more.
### Store
`Discard` store is not persistence user tokens, so `revoke` api is not responded.

`Redis`, `Postgres` and `MYSQL` are supplied by `fns-contrib`, read [doc](https://github.com/aacfactory/fns-contrib/tree/main/authorizations/store) for more.

## API
Encoding, it will return a token.
```go
token, encodingErr := authorizations.Encode(ctx, "userId", userAttributes)
```
Verify current user token in request context, if `@authorization` is true, it will be auto invoked.
```go
verifyErr := authorizations.Verify(ctx)
```
Revoke token.
```go
revokeErr := authorizations.Revoke(ctx, "tokenId")
```
Revoke user tokens.
```go
revokeErr := authorizations.RevokeUserTokens(ctx, "userId")
```