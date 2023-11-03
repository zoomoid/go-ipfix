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
to RFC 7011. Additionally, also supports most other major IPFIX RFCs, namely

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
*/
package ipfix
