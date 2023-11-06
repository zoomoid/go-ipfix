# go-ipfix

[![Go Reference](https://pkg.go.dev/badge/github.com/zoomoid/go-ipfix.svg)](https://pkg.go.dev/github.com/zoomoid/go-ipfix)

go-ipfix is a library for working with IPFIX messages. It supports encoding and decoding of IPFIX messages using a *io.Reader*-style interface.
It complies with RFC 7011, as well as supporting most other major IPFIX RFCs:

- RFC 5103: Bidirectional Flow Export Using IP Flow Information Export (IPFIX)
- RFC 5610: Exporting Type Information for IP Flow Information Export (IPFIX) Information Elements
- RFC 5655: Specification of the IP Flow Information Export (IPFIX) File Format
- RFC 6313: Export of Structured Data in IP Flow Information Export (IPFIX)

## Getting started

- API documentation and examples are available via [pkg.go.dev](https://pkg.go.dev/github.com/zoomoid/go-ipfix)
- The [./addons](./addons) directory contains an implementation of a `ipfix.FieldCache` and `ipfix.TemplateCache` that uses `etcd` for state management

## Contributing

This is a one-person project with (currently) no practical deployment. If you'd like to adopt go-ipfix, require any features or want to
fix any bugs (of which there are probably quite a few remaining), feel free to fork the repository and open a pull request, I promise to
do my best to ensure your efforts make it to the library.
