[![GoDoc](https://godoc.org/github.com/robloxapi/rbxfile?status.png)](https://godoc.org/github.com/robloxapi/rbxfile)

# rbxfile

The rbxfile package handles the decoding, encoding, and manipulation of Roblox
instance data structures.

This package can be used to manipulate Roblox instance trees outside of the
Roblox client. Such data structures begin with a [Root][root] struct. A Root
contains a list of child [Instances][inst], which in turn contain more child
Instances, and so on, forming a tree of Instances. These Instances can be
accessed and manipulated using an API similar to that of Roblox.

Each Instance also has a set of "properties". Each property has a specific
value of a certain [type][type]. Every available type implements the
[Value][value] interface, and is prefixed with "Value".

Root structures can be decoded from and encoded to various formats, including
Roblox's native file formats. The two sub-packages [bin][bin] and [xml][xml]
provide formats for Roblox's binary and XML formats. Root structures can also
be encoded and decoded with the [json][json] package.

Besides decoding from a format, root structures can also be created manually.
The best way to do this is through the [declare][declare] sub-package, which
provides an easy way to generate root structures.

[root]: https://godoc.org/github.com/robloxapi/rbxfile#Root
[inst]: https://godoc.org/github.com/robloxapi/rbxfile#Instance
[type]: https://godoc.org/github.com/robloxapi/rbxfile#Type
[value]: https://godoc.org/github.com/robloxapi/rbxfile#Value
[bin]: https://godoc.org/github.com/robloxapi/rbxfile/bin
[xml]: https://godoc.org/github.com/robloxapi/rbxfile/xml
[json]: https://godoc.org/encoding/json
[declare]: https://godoc.org/github.com/robloxapi/rbxfile/declare
