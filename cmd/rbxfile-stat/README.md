# rbxfile-stat
The **rbxfile-stat** command returns statistics for a roblox file. The following
formats are supported:
- rbxl
- rbxm
- rbxlx
- rbxmx

## Usage
```bash
rbxfile-stat [INPUT] [OUTPUT]
```

Reads a RBXL, RBXM, RBXLX, or RBXMX file from `INPUT`, and writes to `OUTPUT`
statistics for the file.

`INPUT` and `OUTPUT` are paths to files. If `INPUT` is "-" or unspecified, then
stdin is used. If `OUTPUT` is "-" or unspecified, then stdout is used. Warnings
and errors are written to stderr.

## Output
The output is in JSON format with the following structure:

Field             | Type                                   | Description
------------------|----------------------------------------|------------
Format            | [Format](#format)                      | Low-level stats for the file format.
InstanceCount     | int                                    | Actual number of instances.
PropertyCount     | int                                    | Number of individual properties across all instances.
ClassCount        | class -> int                           | Number of instances, per class.
TypeCount         | type -> int                            | Number of properties, per type.
OptionalTypeCount | type -> int                            | Number of properties of the optional type, per inner type.
LargestProperties | array of [PropertyStat](#propertystat) | List of top 20 longest properties. Counts string-like and sequence types.

### Format

Field         | Type   | Description
--------------|--------|------------
XML           | bool   | True if the format is XML, otherwise binary.
Version       | int    | Version of the binary format.
ClassCount    | int    | Number of classes reported by the binary format header.
InstanceCount | int    | Number of instances reported by the binary format header.
Chunks        | int    | Total number of chunks in the binary format.
Chunks        | Chunks | Number of chunks per signature in the binary format.

### PropertyStat

Field         | Type   | Description
--------------|--------|------------
Class         | string | The class name of the instance having this property.
Property      | string | The name of this property.
Type          | string | The type of this property.
Length        | int    | The length of this property.
