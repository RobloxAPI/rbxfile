# rbxfile/bin

[![GoDoc](https://godoc.org/github.com/robloxapi/rbxfile/bin?status.png)](https://godoc.org/github.com/robloxapi/rbxfile/bin)

Package bin implements a decoder and encoder for Roblox's binary file format.

This package registers the formats "rbxl" and "rbxm" to the rbxfile package.

The easiest way to decode and encode files is through the functions
[DeserializePlace][dserp], [SerializePlace][serp], [DeserializeModel][dserm],
and [SerializeModel][serm]. These decode and encode directly between byte
streams and Root structures specified by the rbxfile package. For most
purposes, this is all that is required to read and write Roblox binary files.
Further documentation gives an overview of how the package works internally.

## Overview

A [Serializer][serzr] is used to transform data from byte streams to Root
structures and back. A serializer specifies a decoder and encoder. Both a
decoder and encoder combined is referred to as a "codec".

Codecs transform data between a generic rbxfile.Root structure, and this
package's "format model" structure. Custom codecs can be implemented. For
example, you might wish to decode files normally, but encode them in an
alternative way:

```go
serializer := NewSerializer(nil, CustomEncoder)
```

Custom codecs can be used with a Serializer by implementing the
[Decoder][decr] and [Encoder][encr] interfaces. Both do not need to be
implemented. In the example above, passing nil as an argument causes the
serializer to use the default "[RobloxCodec][roco]", which implements both a
default decoder and encoder. This codec attempts to emulate how Roblox decodes
and encodes its files.

A [FormatModel][fmtm] is the representation of the file format itself, rather
than the data it contains. The FormatModel is like a buffer between the byte
stream and the Root structure. FormatModels can be encoded (and rarely,
decoded) to and from Root structures in multiple ways, which is specified by
codecs. However, there is only one way to encode and decode to and from a byte
stream, which is handled by the FormatModel.

[dserp]: https://godoc.org/github.com/robloxapi/rbxfile/bin#DeserializePlace
[serp]: https://godoc.org/github.com/robloxapi/rbxfile/bin#SerializePlace
[dserm]: https://godoc.org/github.com/robloxapi/rbxfile/bin#DeserializeModel
[serm]: https://godoc.org/github.com/robloxapi/rbxfile/bin#SerializeModel

[rbxfile]: https://godoc.org/github.com/robloxapi/rbxfile
[root]: https://godoc.org/github.com/robloxapi/rbxfile#Root
[serzr]: https://godoc.org/github.com/robloxapi/rbxfile/bin#Serializer
[decr]: https://godoc.org/github.com/robloxapi/rbxfile/bin#Decoder
[encr]: https://godoc.org/github.com/robloxapi/rbxfile/bin#Encoder
[roco]: https://godoc.org/github.com/robloxapi/rbxfile/bin#RobloxCodec
[fmtm]: https://godoc.org/github.com/robloxapi/rbxfile/bin#FormatModel
