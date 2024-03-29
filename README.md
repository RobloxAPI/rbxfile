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
Roblox's native file formats. The two sub-packages [rbxl][rbxl] and
[rbxlx][rbxlx] provide formats for Roblox's binary and XML formats. Root
structures can also be encoded and decoded with the [json][json] package.

Besides decoding from a format, root structures can also be created manually.
The best way to do this is through the [declare][declare] sub-package, which
provides an easy way to generate root structures.

[root]: https://godoc.org/github.com/robloxapi/rbxfile#Root
[inst]: https://godoc.org/github.com/robloxapi/rbxfile#Instance
[type]: https://godoc.org/github.com/robloxapi/rbxfile#Type
[value]: https://godoc.org/github.com/robloxapi/rbxfile#Value
[rbxl]: https://godoc.org/github.com/robloxapi/rbxfile/rbxl
[rbxlx]: https://godoc.org/github.com/robloxapi/rbxfile/rbxlx
[json]: https://godoc.org/encoding/json
[declare]: https://godoc.org/github.com/robloxapi/rbxfile/declare

## Related
The implementation of the binary file format is based largely on the
[RobloxFileSpec][spec] document, a reverse-engineered specification by Gregory
Comer.

Other projects that involve decoding and encoding Roblox files:

- [rbx-fmt](https://github.com/stravant/rbx-fmt): An implementation in C.
- [LibRbxl](https://github.com/GregoryComer/LibRbxl): An implementation in C#.
- [rbx-dom](https://github.com/LPGhatguy/rbx-dom): An implementation in Rust.
- [Roblox-File-Format](https://github.com/CloneTrooper1019/Roblox-File-Format):
  An implementation in C#.

[spec]: https://www.classy-studios.com/Downloads/RobloxFileSpec.pdf
