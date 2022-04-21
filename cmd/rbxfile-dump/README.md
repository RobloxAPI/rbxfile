# rbxfile-dump
The **rbxfile-dump** command dumps the content of binary files (`.rbxl`,
`.rbxm`) in a readable format.

## Usage
```bash
rbxfile-dump [INPUT] [OUTPUT]
```

Reads a binary RBXL or RBXM file from `INPUT`, and dumps a human-readable
representation of the binary format to `OUTPUT`.

`INPUT` and `OUTPUT` are paths to files. If `INPUT` is "-" or unspecified, then
stdin is used. If `OUTPUT` is "-" or unspecified, then stdout is used. Warnings
and errors are written to stderr.

If the command failed to parse a chunk of the input, then the raw bytes of the
chunk will be dumped for further analysis by the user.
