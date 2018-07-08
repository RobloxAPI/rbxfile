To add value type `Foobar`:

- rbxfile
	- `values.go`
		- [ ] Add `TypeFoobar` to Type constants.
		- [ ] In `typeStrings`, map `TypeFoobar` to string `"Foobar"`.
		- [ ] In `valueGenerators`, map `TypeFoobar` to function
		  `newValueFoobar`.
		- [ ] Create `ValueFoobar` type.
			- [ ] Add `ValueFoobar` type with appropriate underlying type.
			- [ ] Implement `newValueFoobar` function (`func() Value`)
			- [ ] Implement `Type() Type` method.
				- Return `TypeFoobar`.
			- [ ] Implement `String() string` method.
				- Return string representation of value that is similar to the
				  results of Roblox's `tostring` function.
			- [ ] Implement `Copy() Value` method.
				- Must return a deep copy of the underlying value.
	- `values_test.go`
		- ...
- declare
	- `declare/type.go`
		- [ ] Add `Foobar` to type constants.
			- Ensure `Foobar` does not conflict with existing identifiers.
		- [ ] In `typeStrings`, map `Foobar` to string `"Foobar"`.
		- [ ] In function `assertValue`, add case `Foobar`.
			- Assert `v` as `rbxfile.ValueFoobar`.
		- [ ] In method `Type.value`, add case `Foobar`.
			- Convert slice of arbitrary values to a `rbxfile.ValueFoobar`.
	- `declare/declare.go`
		- [ ] In function `Property`, document behavior of `Foobar` case in
		  `Type.value` method.
	- `declare/declare_test.go`
		- ...
- json
	- `json/json.go`
		- [ ] In function `ValueToJSONInterface`, add case
		  `rbxfile.ValueFoobar`.
			- Convert `rbxfile.ValueFoobar` to generic JSON interface.
		- [ ] In function `ValueFromJSONInterface`, add case
		  `rbxfile.TypeFoobar`.
			- Convert generic JSON interface to `rbxfile.ValueFoobar`.
- xml
	- `xml/codec.go`
		- [ ] In function `GetCanonType` add case `"foobar"` (lowercase).
			- Returns `"Foobar"`
		- [ ] In method `rdecoder.getValue`, add case `"Foobar"`.
			- Receives `tag *Tag`, must return `rbxfile.ValueFoobar`.
			- `components` can be used to map subtags to value fields.
		- [ ] In method `rencoder.encodeProperty`, add case
		  `rbxfile.ValueFoobar`.
		  	- Returns `*Tag` that is decodable by `rdecoder.getValue`.
		 - [ ] In function `isCanonType`, add case `rbxfile.ValueFoobar`.
- bin
	- `bin/values.go`
		- [ ] Add `TypeFoobar` to type constants.
		- [ ] In `typeStrings`, map `TypeFoobar` to `"Foobar"`.
		- [ ] In `valueGenerators`, map `TypeFoobar` to function
		  `newValueFoobar`.
		- [ ] Create `ValueFoobar` type.
			- [ ] Add `ValueFoobar` with appropriate underlying type.
			- [ ] Implement `newValueFoobar` function (`func() Value`).
			- [ ] Implement `Type() Type` method.
				- Returns `TypeFoobar`.
			- [ ] Implement `ArrayBytes`.
				- Converts a slice of `ValueFoobar` to a slice of bytes.
				- If fields `ValueFoobar` must be interleaved, use
				  `interleaveFields`.
			- [ ] Implement `FromArrayBytes`.
				- Converts a slice of bytes to a slice of `ValueFoobar`.
				- If fields of ValueFoobar` are interleaved, use
				  `deinterleaveFields`.
			- [ ] Implement `Bytes`.
				- Converts a single `ValueFoobar` to a slice of bytes.
			- [ ] Implement `FromBytes`.
				- Converts a slice of bytes to a single `ValueFoobar`.
			- [ ] If fields of `ValueFoobar` must be interleaved, implement
			  `fielder` interface.
				- [ ] Implement `fieldLen`.
					- Returns the byte size of each field.
				- [ ] Implement `fieldSet`.
					- Sets field number `i` using bytes from `b`.
				- [ ] Implement `fieldGet`.
					- Returns field number `i` as a slice of bytes.
	- `bin/codec.go`
		- [ ] In function `decodeValue`, add case `*ValueFoobar`.
			- Converts `*ValueFoobar` to `rbxfile.ValueFoobar`.
		- [ ] In function `encodeValue`, add case `rbxfile.ValueFoobar`.
			- Converts `rbxfile.ValueFoobar` to `*ValueFoobar`.
