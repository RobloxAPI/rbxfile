package bin

import (
	"bytes"
	"errors"
	"github.com/robloxapi/rbxapi"
	"github.com/robloxapi/rbxfile"
	"testing"
)

const goodfile = "<roblox!\x89\xff\r\n\x1a\n\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00END\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
const badfile = "<roblox!\x89\xff\r\n\x1a\n\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00END\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
const encodefile = "<roblox!\x89\xff\r\n\x1a\n\x00\x00\x01\x00\x00\x00\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00INST\x00\x00\x00\x00\x1e\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\r\x00\x00\x00EncodeSuccess\x00\x01\x00\x00\x00\x00\x00\x00\x00END\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

var badroot = &rbxfile.Root{}
var goodroot = &rbxfile.Root{
	Instances: []*rbxfile.Instance{
		rbxfile.NewInstance("EncodeSuccess", nil),
	},
}

type testDecoder struct{}

func (testDecoder) Decode(model *FormatModel, api *rbxapi.API) (root *rbxfile.Root, err error) {
	if model.TypeCount == 0 {
		return nil, errors.New("decode fail")
	}

	root = &rbxfile.Root{
		Instances: []*rbxfile.Instance{
			rbxfile.NewInstance("DecodeSuccess", nil),
		},
	}

	return
}

type testEncoder struct{}

func (testEncoder) Encode(root *rbxfile.Root, api *rbxapi.API) (model *FormatModel, err error) {
	if len(root.Instances) == 0 {
		return nil, errors.New("encode fail")
	}

	model = new(FormatModel)
	model.TypeCount = 1
	model.InstanceCount = 1
	model.Chunks = []Chunk{
		&ChunkInstance{
			ClassName:   "EncodeSuccess",
			InstanceIDs: []int32{0},
		},
		&ChunkEnd{},
	}

	return
}

func TestNewSerializer(t *testing.T) {
	ser := NewSerializer(nil, nil)
	if _, ok := ser.Decoder.(RobloxCodec); !ok {
		t.Error("unexpected serializer decoder (not RobloxCodec)")
	}
	if _, ok := ser.Encoder.(RobloxCodec); !ok {
		t.Error("unexpected serializer encoder (not RobloxCodec)")
	}

	ser = NewSerializer(testDecoder{}, nil)
	if _, ok := ser.Decoder.(testDecoder); !ok {
		t.Error("unexpected serializer decoder (not testDecoder)")
	}
	if _, ok := ser.Encoder.(RobloxCodec); !ok {
		t.Error("unexpected serializer encoder (not RobloxCodec)")
	}

	ser = NewSerializer(nil, testEncoder{})
	if _, ok := ser.Decoder.(RobloxCodec); !ok {
		t.Error("unexpected serializer decoder (not RobloxCodec)")
	}
	if _, ok := ser.Encoder.(testEncoder); !ok {
		t.Error("unexpected serializer encoder (not testEncoder)")
	}

	ser = NewSerializer(testDecoder{}, testEncoder{})
	if _, ok := ser.Decoder.(testDecoder); !ok {
		t.Error("unexpected serializer decoder (not testDecoder)")
	}
	if _, ok := ser.Encoder.(testEncoder); !ok {
		t.Error("unexpected serializer encoder (not testEncoder)")
	}
}

func TestSerializer_Deserialize(t *testing.T) {
	ser := Serializer{}

	if _, err := ser.Deserialize(nil, nil); err == nil {
		t.Error("expected error (no decoder)")
	}

	ser.Decoder = testDecoder{}

	if _, err := ser.Deserialize(nil, nil); err == nil {
		t.Error("expected error (format parsing)")
	}

	buf := bytes.NewBufferString(badfile)
	if _, err := ser.Deserialize(buf, nil); err == nil {
		t.Error("expected error (decoding)")
	}

	buf = bytes.NewBufferString(goodfile)
	if root, err := ser.Deserialize(buf, nil); err != nil {
		t.Error("unexpected error", err)
	} else if len(root.Instances) == 0 || root.Instances[0].ClassName != "DecodeSuccess" {
		t.Error("unexpected Root")
	}
}

func TestSerializer_Serialize(t *testing.T) {
	ser := Serializer{}

	if err := ser.Serialize(nil, nil, nil); err == nil {
		t.Error("expected error (no encoder)")
	}

	ser.Encoder = testEncoder{}

	if err := ser.Serialize(nil, nil, badroot); err == nil {
		t.Error("expected error (encoding data)")
	}

	if err := ser.Serialize(nil, nil, goodroot); err == nil {
		t.Error("expected error (encoding format)")
	}

	var buf bytes.Buffer
	if err := ser.Serialize(&buf, nil, goodroot); err != nil {
		t.Error("unexpected error", err)
	}

	if buf.String() != encodefile {
		t.Error("unexpected file content")
	}
}

func TestSerializeFunc(t *testing.T) {
	DeserializePlace(nil, nil)
	SerializePlace(nil, nil, nil)
	DeserializeModel(nil, nil)
	SerializeModel(nil, nil, nil)
}

func TestFormat(t *testing.T) {
	f := format{
		name:       "TestName",
		magic:      "TestMagic",
		serializer: NewSerializer(testDecoder{}, testEncoder{}),
	}

	if f.Name() != "TestName" {
		t.Error("unexpected result from Name")
	}

	if f.Magic() != "TestMagic" {
		t.Error("unexpected result from Magic")
	}

	if root, err := f.Decode(bytes.NewBufferString(goodfile), nil); err != nil {
		t.Error("unexpected error", err)
	} else if len(root.Instances) == 0 || root.Instances[0].ClassName != "DecodeSuccess" {
		t.Error("unexpected Root")
	}

	var buf bytes.Buffer
	if err := f.Encode(&buf, nil, goodroot); err != nil {
		t.Error("unexpected error", err)
	}

	if buf.String() != encodefile {
		t.Error("unexpected file content")
	}
}
