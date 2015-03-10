[![GoDoc](https://godoc.org/github.com/RobloxAPI/rbxfile?status.png)](https://godoc.org/github.com/RobloxAPI/rbxfile)

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
Roblox's native file formats. This is done with the [Decode][dec] and
[Encode][enc] functions. In order to use these functions, one or more formats
must be registered. Usually, this occurs as a side-effect of importing a
package that implements the format. For example, the [bin][bin] sub-package,
which implements Roblox's binary format, registers the formats "rbxl" and
"rbxm" when imported:

```go
import _ "github.com/robloxapi/rbxfile/bin"
```

Besides decoding from a format, root structures can also be created manually.
The best way to do this is through the [declare][declare] sub-package, which
provides an easy way to generate root structures.

[root]: https://godoc.org/github.com/RobloxAPI/rbxfile#Root
[inst]: https://godoc.org/github.com/RobloxAPI/rbxfile#Instance
[type]: https://godoc.org/github.com/RobloxAPI/rbxfile#Type
[value]: https://godoc.org/github.com/RobloxAPI/rbxfile#Value
[dec]: https://godoc.org/github.com/RobloxAPI/rbxfile#Decode
[enc]: https://godoc.org/github.com/RobloxAPI/rbxfile#Encode
[bin]: https://godoc.org/github.com/RobloxAPI/rbxfile/bin
[declare]: https://godoc.org/github.com/RobloxAPI/rbxfile/declare
