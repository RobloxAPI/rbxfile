# rbxfile-dcomp
The **rbxfile-dcomp** command rewrites the content of binary files (`.rbxl`,
`.rbxm`) with decompressed chunks, allowing the content of such files to be
analyzed more easily.

## Usage
```bash
rbxfile-dcomp [INPUT] [OUTPUT]
```

Reads a binary RBXL or RBXM file from `INPUT`, and writes to `OUTPUT` the same
file, but with uncompressed chunks.

`INPUT` and `OUTPUT` are paths to files. If `INPUT` is "-" or unspecified, then
stdin is used. If `OUTPUT` is "-" or unspecified, then stdout is used. Warnings
and errors are written to stderr.
