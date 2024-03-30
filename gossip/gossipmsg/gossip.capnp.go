// Code generated by capnpc-go. DO NOT EDIT.

package gossipmsg

import (
	capnp "capnproto.org/go/capnp/v3"
	text "capnproto.org/go/capnp/v3/encoding/text"
	schemas "capnproto.org/go/capnp/v3/schemas"
	strconv "strconv"
)

type Gossip capnp.Struct
type Gossip_signature Gossip
type Gossip_data Gossip
type Gossip_Which uint16

const (
	Gossip_Which_signature Gossip_Which = 0
	Gossip_Which_data      Gossip_Which = 1
)

func (w Gossip_Which) String() string {
	const s = "signaturedata"
	switch w {
	case Gossip_Which_signature:
		return s[0:9]
	case Gossip_Which_data:
		return s[9:13]

	}
	return "Gossip_Which(" + strconv.FormatUint(uint64(w), 10) + ")"
}

// Gossip_TypeID is the unique identifier for the type Gossip.
const Gossip_TypeID = 0xf72bafaeff08c61a

func NewGossip(s *capnp.Segment) (Gossip, error) {
	st, err := capnp.NewStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 3})
	return Gossip(st), err
}

func NewRootGossip(s *capnp.Segment) (Gossip, error) {
	st, err := capnp.NewRootStruct(s, capnp.ObjectSize{DataSize: 8, PointerCount: 3})
	return Gossip(st), err
}

func ReadRootGossip(msg *capnp.Message) (Gossip, error) {
	root, err := msg.Root()
	return Gossip(root.Struct()), err
}

func (s Gossip) String() string {
	str, _ := text.Marshal(0xf72bafaeff08c61a, capnp.Struct(s))
	return str
}

func (s Gossip) EncodeAsPtr(seg *capnp.Segment) capnp.Ptr {
	return capnp.Struct(s).EncodeAsPtr(seg)
}

func (Gossip) DecodeFromPtr(p capnp.Ptr) Gossip {
	return Gossip(capnp.Struct{}.DecodeFromPtr(p))
}

func (s Gossip) ToPtr() capnp.Ptr {
	return capnp.Struct(s).ToPtr()
}

func (s Gossip) Which() Gossip_Which {
	return Gossip_Which(capnp.Struct(s).Uint16(0))
}
func (s Gossip) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Gossip) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Gossip) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Gossip) Id() ([]byte, error) {
	p, err := capnp.Struct(s).Ptr(0)
	return []byte(p.Data()), err
}

func (s Gossip) HasId() bool {
	return capnp.Struct(s).HasPtr(0)
}

func (s Gossip) SetId(v []byte) error {
	return capnp.Struct(s).SetData(0, v)
}

func (s Gossip) Signature() Gossip_signature { return Gossip_signature(s) }

func (s Gossip) SetSignature() {
	capnp.Struct(s).SetUint16(0, 0)
}

func (s Gossip_signature) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Gossip_signature) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Gossip_signature) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Gossip_signature) Signer() ([]byte, error) {
	p, err := capnp.Struct(s).Ptr(1)
	return []byte(p.Data()), err
}

func (s Gossip_signature) HasSigner() bool {
	return capnp.Struct(s).HasPtr(1)
}

func (s Gossip_signature) SetSigner(v []byte) error {
	return capnp.Struct(s).SetData(1, v)
}

func (s Gossip_signature) Signature() ([]byte, error) {
	p, err := capnp.Struct(s).Ptr(2)
	return []byte(p.Data()), err
}

func (s Gossip_signature) HasSignature() bool {
	return capnp.Struct(s).HasPtr(2)
}

func (s Gossip_signature) SetSignature(v []byte) error {
	return capnp.Struct(s).SetData(2, v)
}

func (s Gossip) Data() Gossip_data { return Gossip_data(s) }

func (s Gossip) SetData() {
	capnp.Struct(s).SetUint16(0, 1)
}

func (s Gossip_data) IsValid() bool {
	return capnp.Struct(s).IsValid()
}

func (s Gossip_data) Message() *capnp.Message {
	return capnp.Struct(s).Message()
}

func (s Gossip_data) Segment() *capnp.Segment {
	return capnp.Struct(s).Segment()
}
func (s Gossip_data) Data() ([]byte, error) {
	p, err := capnp.Struct(s).Ptr(1)
	return []byte(p.Data()), err
}

func (s Gossip_data) HasData() bool {
	return capnp.Struct(s).HasPtr(1)
}

func (s Gossip_data) SetData(v []byte) error {
	return capnp.Struct(s).SetData(1, v)
}

// Gossip_List is a list of Gossip.
type Gossip_List = capnp.StructList[Gossip]

// NewGossip creates a new list of Gossip.
func NewGossip_List(s *capnp.Segment, sz int32) (Gossip_List, error) {
	l, err := capnp.NewCompositeList(s, capnp.ObjectSize{DataSize: 8, PointerCount: 3}, sz)
	return capnp.StructList[Gossip](l), err
}

// Gossip_Future is a wrapper for a Gossip promised by a client call.
type Gossip_Future struct{ *capnp.Future }

func (f Gossip_Future) Struct() (Gossip, error) {
	p, err := f.Future.Ptr()
	return Gossip(p.Struct()), err
}
func (p Gossip_Future) Signature() Gossip_signature_Future { return Gossip_signature_Future{p.Future} }

// Gossip_signature_Future is a wrapper for a Gossip_signature promised by a client call.
type Gossip_signature_Future struct{ *capnp.Future }

func (f Gossip_signature_Future) Struct() (Gossip_signature, error) {
	p, err := f.Future.Ptr()
	return Gossip_signature(p.Struct()), err
}
func (p Gossip_Future) Data() Gossip_data_Future { return Gossip_data_Future{p.Future} }

// Gossip_data_Future is a wrapper for a Gossip_data promised by a client call.
type Gossip_data_Future struct{ *capnp.Future }

func (f Gossip_data_Future) Struct() (Gossip_data, error) {
	p, err := f.Future.Ptr()
	return Gossip_data(p.Struct()), err
}

const schema_fbd8d724be65e33e = "x\xda\x8cQ1K#Q\x18\x9cyo\xef\x92\xe2\x96" +
	"d\xd9\\\x13\xee\xacb\x91\x98\x18%\x08\x12\xc1\xa4\x09" +
	"\"Z\xe4Ka\xbf\x98e\xd9\xc2$d#\x82\x95\xad" +
	"\x9d\x8d`mai*-\x05\x0b\xfd\x03\x16\xda[[" +
	"[\x88>\xd9\x04\xa2`\x04\xab\xc7\xcc\x1b\xe6\x9ba\xd2" +
	"\x97uk\xd1\xee*(\xf9\xf7\xeb\xb79z\xf2\xef6" +
	"\x0e\xb6N \xb3\xa4\xc9\xde&\xcd\xf9p\xee\x19\x7fu" +
	"\x82@\xe5?\xf7\x09\xbay\xee\x81f\xa5\xf8\xda<s" +
	"K\xa7\xd3\xa5\x87l\xc5\xd2c\xd6\xf0\xe9SfH\xb3" +
	"\xfa\xe8_\xe5\xee\x1f^\xd0\xd0\x09\x05T.X\xa0{" +
	"\xc3\x04\xe0^s\x88\x92\x09\xbaQ\x14\xf6\xca\x815z" +
	"w\xa2\xa0<f\xe6\xb7\xbd^\xa7W]\x1b\x83(\x0c" +
	"j\x1do\xb0\xdb\xf7%\xa9\xad43$\xe0\xe4\xab\x80" +
	"\xe44eA\xd1\xa1\xcaP\x01N\xa9\x05HQS\x96" +
	"\x15kQ\x18t\xfc>m(\xda\xa0\x89a\xec\x02\xfa" +
	"\x13\xeeg\x01\xda\xde\x80\x9eX\xe3\xdb\x1ap\xec\x02 " +
	"IM\xc9(\xa6\xda\xde\xc0\xfbb\xa8\xbf3L\xc5\xa8" +
	"I\xca\x1fm\x01V\xdc\xa4\x91\x05\xa4\xae)\x9b\x8a6" +
	"\x8d!?\xd6q\xd6[P\xb6z\x8b\xc9\xc9\x0e\xceR" +
	"\x01J\x87\xedi\xd5Fq\xde\x03\x00\x00\xff\xff\xf9S" +
	"\x86\xe6"

func RegisterSchema(reg *schemas.Registry) {
	reg.Register(&schemas.Schema{
		String: schema_fbd8d724be65e33e,
		Nodes: []uint64{
			0x9856804bd365ed90,
			0xa22d13a650fd2c3b,
			0xf72bafaeff08c61a,
		},
		Compressed: true,
	})
}
