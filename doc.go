/*
Copyright 2023 Alexander Bartolomey (github@alexanderbartolomey.de)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package for working with IPFIX messages. Supports decoding and encoding from and to IPFIX according
to RFC 7011.

# Overview

IPFIX message format is defined in RFC 7011. go-ipfix implements decoding and encoding of messages adhering
to that RFC. Additionally, most other major IPFIX RFCs are also supported feature-wise, namely

- RFC 5103: Bidirectional Flow Export Using IP Flow Information Export (IPFIX)

- RFC 5610: Exporting Type Information for IP Flow Information Export (IPFIX) Information Elements

- RFC 5655: Specification of the IP Flow Information Export (IPFIX) File Format

- RFC 6313: Export of Structured Data in IP Flow Information Export (IPFIX)

Below are some examples of how some common use-cases of this library may look like.

# Historical Background

This library was factored out of a 2023 master's thesis' codebase for working with IPFIX flow records.
The ipfix package works on its own, and was used for implementing collectors and further processing
tools for IPFIX flow records, in particular in combination with enterprise-specific information elements
for proprietary flow information, and structured data types, which are prominent when using
yaf (https://tools.netsa.cert.org/yaf/) in combination with DPI information.

When factoring out, the Decode API was overhauled to mirror Go's io.Reader style. Additionally,
some high-level types such as TemplateSet, DataSet, and OptionsTemplateSet were cleaned up.

Also, TCP and UDP listeners were not originally part of the ipfix module. While UDP is much simpler to implement
as one UDP packet generally corresponds to one IPFIX message, and such a listener is easy to implement
using Go's rich net.PacketConn tooling, IPFIX's prescription of how collection via a _single long-lived_ TCP connection
works is a bit more involved and we decided to include our contribution of such a TCP listener in
this package. The below example shows how to use it, and in particular shows how the TCP and UDP listener API
is identical.

# Data Structures

IPFIX messages are nested to contain as much possible data in a single message as possible.
On a top-level, an IPFIX message contains 3 kinds of typed sets of records, namely template sets, data sets, and
options template sets. These sets have a header-like data structure at the start denoting the "ID" of the set.
In the case of template records, the ID is expected to be 2, for options template sets it is 3, and data sets are freely
taken from 255 up to 65535. The IDs of data sets are associated with IDs defined in previously sent template records.

Each set contains one or more records of the same type, i.e., template records, data records, options template records.

Each record contains one or more fields, whose semantics are defined either in the standard registry 0 of IANA-IPFIX, or
vendor-specific/proprietary are Information Elements.

Values of such fields are typed according to RFC 7011 or RFC 6313 (which defines higher-level data types). In go-ipfix,
all those data types implement the DataType interface. For more details on the data types consult the corresponding RFCs.

Notably, RFC 6313 defines mechanisms for (recursively) nesting records in data records. The SubTemplateList and SubTemplateMultiList
data types allow for nesting data records created from *possibly different* templates than the containing data set, thus creating
a tree-like structure, where records can be nested up to a certain extent (not exceeding the maximum message size of 2^16-1 bytes minus a couple of bytes for headers).

The coupling of templates and records (read: data semantics are detached from the actual data) requires a stateful management of
the templates such that incoming data records can be decoded if and only if the corresponding template was received before. RFC 7011
prescribes that decoders (collectors) may store records not able to decode due to absense of the template up to a certain point if e.g.
asymmetric paths create such a race condition. go-ipfix *does not* implement such a system, but the error returned from the Decoder if
a template is not known can be used to queue up such messages and work off that queue at any later point.
*/
package ipfix
