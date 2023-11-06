# go-ipfix/addons/etcd

`go-ipfix/addons/etcd` is a FieldCache/TemplateCache implementation using etcd under the hood for *distributed* and *strongly-consistent* management of templates and fields.
It may be used to scale-out a collector and serves as an example of how to improve state management for IPFIX connections.

