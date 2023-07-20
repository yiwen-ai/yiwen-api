package content

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

type AttrKind int

const (
	Vnull AttrKind = iota
	Vbool
	Vint64
	Vfloat64
	Vstring
)

func (k AttrKind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return kindNames[0]
}

var kindNames = []string{
	Vnull:    "null",
	Vbool:    "bool",
	Vint64:   "int64",
	Vfloat64: "float64",
	Vstring:  "string",
}

type AttrValue struct {
	kind AttrKind
	v    any
}

func Bool(v bool) AttrValue {
	return AttrValue{kind: Vbool, v: v}
}

func Int64(v int64) AttrValue {
	return AttrValue{kind: Vint64, v: v}
}

func Float64(v float64) AttrValue {
	return AttrValue{kind: Vfloat64, v: v}
}

func String(v string) AttrValue {
	return AttrValue{kind: Vstring, v: v}
}

func (v *AttrValue) Kind() AttrKind {
	if v == nil {
		return Vnull
	}
	return v.kind
}

func (v *AttrValue) Is(k AttrKind) bool {
	if v == nil {
		return false
	}
	return v.kind == k
}

func (v AttrValue) ToBool() bool {
	x, _ := v.v.(bool)
	return x
}

func (v AttrValue) ToInt64() int64 {
	x, _ := v.v.(int64)
	return x
}

func (v AttrValue) ToFloat64() float64 {
	x, _ := v.v.(float64)
	return x
}

func (v AttrValue) ToString() string {
	x, _ := v.v.(string)
	return x
}

func (v AttrValue) ToAny() any {
	return v.v
}

func (v AttrValue) GoString() string {
	return fmt.Sprintf("%#v", v.ToAny())
}

func (v AttrValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.v)
}

func (v AttrValue) MarshalCBOR() ([]byte, error) {
	return cbor.Marshal(v.v)
}

func (v *AttrValue) UnmarshalCBOR(data []byte) error {
	if v == nil {
		return errors.New("content.AttrValue: UnmarshalCBOR on nil pointer")
	}
	if err := cbor.Unmarshal(data, &v.v); err != nil {
		return err
	}

	switch v.v.(type) {
	case nil:
		v.kind = Vnull
	case bool:
		v.kind = Vbool
	case int64:
		v.kind = Vint64
	case float64:
		v.kind = Vfloat64
	case string:
		v.kind = Vstring
	default:
		v.kind = Vnull
		v.v = nil
		return fmt.Errorf("content.AttrValue: unknown type %T", v.v)
	}

	return nil
}

func (v *AttrValue) UnmarshalJSON(data []byte) error {
	if v == nil {
		return errors.New("content.AttrValue: UnmarshalJSON on nil pointer")
	}
	if err := json.Unmarshal(data, &v.v); err != nil {
		return err
	}

	switch v.v.(type) {
	case nil:
		v.kind = Vnull
	case bool:
		v.kind = Vbool
	case int64:
		v.kind = Vint64
	case float64:
		v.kind = Vfloat64
	case string:
		v.kind = Vstring
	default:
		v.kind = Vnull
		v.v = nil
		return fmt.Errorf("content.AttrValue: unknown type %T", v.v)
	}

	return nil
}
