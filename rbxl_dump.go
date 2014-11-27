package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/bkaradzic/go-lz4"
	"io"
	"math"
	"os"
)

// Interleave transforms an array of bytes by interleaving them based on a
// given size. The size must be a divisor of the array length.
//
// The array is divided into groups, each of length `size`. The nth elements
// of each group are then moved so that they are group together. For example:
//     Original:    abcd1234
//     Interleaved: a1b2c3d4
func interleave(bytes []byte, size int) {
	if size <= 0 {
		panic("size must be greater than 0")
	}
	if len(bytes)%size != 0 {
		panic("size must be a divisor of array length")
	}

	// Matrix transpose algorithm
	cols := size
	rows := len(bytes) / size
	if rows == cols {
		for r := 0; r < rows; r++ {
			for c := 0; c < r; c++ {
				bytes[r*cols+c], bytes[c*cols+r] = bytes[c*cols+r], bytes[r*cols+c]
			}
		}
	} else {
		tmp := make([]byte, len(bytes))
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				tmp[c*rows+r] = bytes[r*cols+c]
			}
		}
		for i, b := range tmp {
			bytes[i] = b
		}
	}
}

func deinterleave(bytes []byte, size int) {
	if size <= 0 {
		panic("size must be greater than 0")
	}
	if len(bytes)%size != 0 {
		panic("size must be a divisor of array length")
	}

	interleave(bytes, len(bytes)/size)
}

func encodeZigzag(n int32) uint32 {
	//	if n < 0 {
	//		return uint32(-2*n - 1)
	//	} else {
	//		return uint32(n) * 2
	//	}
	return uint32((n << 1) ^ (n >> 31))
}

func decodeZigzag(n uint32) int32 {
	//	if n%2 == 0 {
	//		return int32(n / 2)
	//	} else {
	//		return -int32(n/2) - 1
	//	}
	return int32((n >> 1) ^ uint32((int32(n&1)<<31)>>31))
}

// Circular left shift.
func rotl(n uint32) uint32 {
	return (n << 1) | (n >> 31)
}

// Circular right shift.
func rotr(n uint32) uint32 {
	return (n >> 1) | (n << 31)
}

// Binary32 with sign at LSB instead of MSB.
func encodeRobloxFloat(n float32) uint32 {
	return rotl(math.Float32bits(n))
}

func decodeRobloxFloat(n uint32) float32 {
	return math.Float32frombits(rotr(n))
}

func expect(r io.Reader, b []byte) {
	fmt.Printf("\tEXPECT:%#v\n", string(b))
	v := make([]byte, len(b))
	r.Read(v)
	if !bytes.Equal(v, b) {
		panic("expected `" + string(b) + "`, got `" + string(v) + "`")
	}
}

func readString(r io.Reader) string {
	var length uint32
	binary.Read(r, binary.LittleEndian, &length)
	str := make([]byte, length)
	r.Read(str)
	return string(str)
}

func readChunk(r io.Reader, tag []byte) *bytes.Reader {
	//expect(r, tag)

	fmt.Printf("\tLZ4 DATA:\n")

	var cmpLen uint32
	binary.Read(r, binary.LittleEndian, &cmpLen)
	fmt.Printf("\t\tCOMPRESSED LENGTH:%d\n", cmpLen)

	var dcmpLen uint32
	binary.Read(r, binary.LittleEndian, &dcmpLen)
	fmt.Printf("\t\tDECOMPRESSED LENGTH:%d\n", dcmpLen)

	var reserved uint32
	binary.Read(r, binary.LittleEndian, &reserved)
	fmt.Printf("\t\tRESERVED:%d\n", reserved)

	dcmpData := make([]byte, dcmpLen)
	// If compressed length is 0, then the data is not compressed.
	if cmpLen == 0 {
		fmt.Printf("\t\tDATA NOT COMPRESSED\n")
		r.Read(dcmpData)
	} else {
		// Prepare compressed data for reading by lz4, which requires the
		// uncompressed length before the compressed data.
		cmpData := make([]byte, cmpLen+4)
		binary.LittleEndian.PutUint32(cmpData, dcmpLen)
		r.Read(cmpData[4:])
		// ERROR: Malformed data ([true decompressed length] != [given decompressed length])
		lz4.Decode(dcmpData, cmpData)
	}

	//fmt.Printf("\t\tDECOMPRESSED DATA:% 3x\n", dcmpData)
	return bytes.NewReader(dcmpData)
}

func readInstChunk(r io.Reader) {
	fmt.Println("\tTYPE INST")
	dr := readChunk(r, []byte{0x49, 0x4E, 0x53, 0x54}) // INST

	var typeID uint32
	binary.Read(dr, binary.LittleEndian, &typeID)
	fmt.Printf("\tTYPE ID:%d\n", typeID)

	typeName := readString(dr)
	fmt.Printf("\tTYPE NAME:%#v\n", typeName)

	var moreData uint8
	binary.Read(dr, binary.LittleEndian, &moreData)
	additionalData := moreData != 0
	fmt.Printf("\tHAS ADDITIONAL DATA:%t\n", additionalData)

	var nObjs uint32
	binary.Read(dr, binary.LittleEndian, &nObjs)
	fmt.Printf("\tREFERENT ARRAY LENGTH:%d\n", nObjs)

	refsRaw := make([]byte, nObjs*4)
	dr.Read(refsRaw)
	deinterleave(refsRaw, 4)
	rr := bytes.NewReader(refsRaw)

	refs := make([]int32, nObjs)
	for i := range refs {
		var n uint32
		binary.Read(rr, binary.BigEndian, &n)
		refs[i] = decodeZigzag(n)
	}
	// Each entry is relative to the previous, convert to actual referents.
	for i := 1; i < len(refs); i++ {
		refs[i] = refs[i-1] + refs[i]
	}

	fmt.Printf("\tREFERENT ARRAY:\n")
	for _, ref := range refs {
		fmt.Printf("\t\tREF:%d\n", ref)
	}

	if additionalData {
		addData := make([]byte, nObjs)
		dr.Read(addData)
		fmt.Printf("\tADDITIONAL DATA:%v\n", addData)
	}
}

func readPropChunk(r io.Reader) {
	fmt.Println("\tTYPE PROP")
	dr := readChunk(r, []byte{0x50, 0x52, 0x4F, 0x50}) // PROP

	var typeID uint32
	binary.Read(dr, binary.LittleEndian, &typeID)
	fmt.Printf("\tTYPE ID:%d\n", typeID)

	propName := readString(dr)
	fmt.Printf("\tNAME:%#v\n", propName)

	var dataType uint8
	binary.Read(dr, binary.LittleEndian, &dataType)
	fmt.Printf("\tDATA TYPE:%d\n", dataType)
}

func readPrntChunk(r io.Reader) {
	fmt.Println("\tTYPE PRNT")
	dr := readChunk(r, []byte{0x50, 0x52, 0x4E, 0x54}) // PRNT

	var version uint8
	binary.Read(dr, binary.LittleEndian, &version)
	fmt.Printf("\tVERSION:%d\n", version)

	var nObjs uint32
	binary.Read(dr, binary.LittleEndian, &nObjs)
	fmt.Printf("\tOBJECT COUNT:%d\n", nObjs)

	refsRaw := make([]byte, nObjs*4)
	dr.Read(refsRaw)
	deinterleave(refsRaw, 4)
	rr := bytes.NewReader(refsRaw)

	refs := make([]int32, nObjs)
	for i := range refs {
		var n uint32
		binary.Read(rr, binary.BigEndian, &n)
		refs[i] = decodeZigzag(n)
	}
	// Each entry is relative to the previous, convert to actual referents.
	for i := 1; i < len(refs); i++ {
		refs[i] = refs[i-1] + refs[i]
	}

	fmt.Printf("\tREFERENT ARRAY:\n")
	for _, ref := range refs {
		fmt.Printf("\t\tREF:%d\n", ref)
	}

	parentsRaw := make([]byte, nObjs*4)
	dr.Read(parentsRaw)
	deinterleave(parentsRaw, 4)
	pr := bytes.NewReader(parentsRaw)

	parents := make([]int32, nObjs)
	for i := range parents {
		var n uint32
		binary.Read(pr, binary.BigEndian, &n)
		parents[i] = decodeZigzag(n)
	}
	// Each entry is relative to the previous, convert to actual referents.
	for i := 1; i < len(parents); i++ {
		parents[i] = parents[i-1] + parents[i]
	}

	fmt.Printf("\tPARENT ARRAY:\n")
	for _, ref := range parents {
		fmt.Printf("\t\tPARENT:%d\n", ref)
	}
}

func readEndChunk(r io.Reader) {
	fmt.Println("\tTYPE END")
	dr := readChunk(r, []byte{0x45, 0x4E, 0x44, 0x00}) // END\0

	expect(dr, []byte("</roblox>"))

	if dr.Len() > 0 {
		panic("invalid end chunk")
	}
}

func main() {
	defer func() {
		if msg := recover(); msg != nil {
			fmt.Println("PANIC: ", msg)
		}
	}()

	f, err := os.Open("test.rbxl")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// read header
	fmt.Println("HEADER:")
	expect(f, []byte("<roblox!\x89\xff\r\n\x1a\n"))

	var version uint16
	binary.Read(f, binary.LittleEndian, &version)
	fmt.Printf("VERSION:%d\n", version)

	var numTypes uint32
	binary.Read(f, binary.LittleEndian, &numTypes)
	fmt.Printf("NUM TYPES:%d\n", numTypes)

	var numObjects uint32
	binary.Read(f, binary.LittleEndian, &numObjects)
	fmt.Printf("NUM OBJECTS:%d\n", numObjects)

	var reserved uint64
	binary.Read(f, binary.LittleEndian, &reserved)
	fmt.Printf("RESERVED:%d\n", reserved)

loop:
	for i := 0; true; i++ {
		fmt.Printf("CHUNK %d:\n", i)

		tag := make([]byte, 4)
		if _, err := f.Read(tag); err == io.EOF {
			break
		}

		switch string(tag) {
		case "INST":
			readInstChunk(f)
		case "PROP":
			readPropChunk(f)
		case "PRNT":
			readPrntChunk(f)
		case "END\000":
			readEndChunk(f)
			break loop
		default:
			panic("unexpected chunk `" + string(tag) + "`")
		}
	}

	fmt.Println("READ COMPLETE")
}
