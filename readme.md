# Not a Framework
`nf` is a Go package designed to aid us in building HTTP services at IMQS.
It is IMQS specific - tying into our authentication service, our logging, etc.

`nf/nfdb` contains helpers for SQL databases
`nf/nftest` contains helpers for building unit tests

See godoc documentation at [nf](https://godoc.org/github.com/IMQS/nf), [nfdb](https://godoc.org/github.com/IMQS/nf/nfdb), [nftest](https://godoc.org/github.com/IMQS/nf/nftest)

## Examples
See [github.com/IMQS/gostarter](https://github.com/IMQS/gostarter) for a real-world example.

## HTTP Handlers
The functions [Handle](https://godoc.org/github.com/IMQS/nf#Handle) and [HandleAuthenticated](https://godoc.org/github.com/IMQS/nf#HandleAuthenticated)
create an HTTP API entrypoint handler. Your handler function is free to panic inside these handlers, an the panic will be
interpreted by the wrapper, and an appropriate HTTP error code sent back to the caller.

If you call `nf.Panic(403, "Operation not allowed")`, then the caller will receive the intended response.