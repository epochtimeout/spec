package spec

import (
	"errors"
	"fmt"
	"sync"

	"github.com/complexl/library/u128"
	"github.com/complexl/library/u256"
)

const BufferSize = 4096

var encoderPool = &sync.Pool{
	New: func() interface{} {
		return newEncoder(nil)
	},
}

// Encoder encodes values.
type Encoder struct {
	buf  []byte
	err  error      // encoding failed
	data encodeData // last written data, must be consumed before writing next data

	objects  objectStack
	elements listStack    // stack of list element tables
	fields   messageStack // stack of message field tables

	// preallocated
	_objects  [16]objectEntry
	_elements [128]listElement
	_fields   [128]messageField
}

// NewEncoder returns a new encoder with a default buffer.
//
// Usually, it is better to use Encode(obj) and EncodeTo(obj, buf), than to construct
// a new encoder directly. These methods internally use an encoder pool.
func NewEncoder() *Encoder {
	buf := make([]byte, 0, BufferSize)
	return newEncoder(buf)
}

// NewEncoderBuffer returns a new encoder with a buffer.
//
// Usually, it is better to use Encode(obj) and EncodeTo(obj, buf), than to construct
// a new encoder directly. These methods internally use an encoder pool.
func NewEncoderBuffer(buf []byte) *Encoder {
	return newEncoder(buf)
}

func newEncoder(buf []byte) *Encoder {
	e := &Encoder{
		buf:  buf[:0],
		data: encodeData{},
	}

	e.objects.stack = e._objects[:0]
	e.elements.stack = e._elements[:0]
	e.fields.stack = e._fields[:0]
	return e
}

// End ends writing, returns the result bytes, and resets the encoder.
func (e *Encoder) End() ([]byte, error) {
	switch {
	case e.err != nil:
		return nil, e.err
	case e.objects.len() > 0:
		return nil, fmt.Errorf("end: incomplete object, nested stack size=%d", e.objects.len())
	}

	// pop data
	data := e.popData()

	// return and reset
	b := e.buf[data.start:data.end]
	e.Reset()
	return b, nil
}

// Init resets the encoder and sets its buffer.
func (e *Encoder) Init(b []byte) {
	e.Reset()
	e.buf = b[:0]
}

// Reset clears the encoder and nils buffer.
func (e *Encoder) Reset() {
	e.buf = nil
	e.err = nil
	e.data = encodeData{}

	e.objects.reset()
	e.elements.reset()
	e.fields.reset()
}

// Primitive

func (e *Encoder) Nil() error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeNil(e.buf)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

func (e *Encoder) Bool(v bool) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeBool(e.buf, v)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

func (e *Encoder) Byte(v byte) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeByte(e.buf, v)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

func (e *Encoder) Int32(v int32) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeInt32(e.buf, v)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

func (e *Encoder) Int64(v int64) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeInt64(e.buf, v)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

func (e *Encoder) Uint32(v uint32) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeUint32(e.buf, v)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

func (e *Encoder) Uint64(v uint64) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeUint64(e.buf, v)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

// U128/U256

func (e *Encoder) U128(v u128.U128) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeU128(e.buf, v)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

func (e *Encoder) U256(v u256.U256) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeU256(e.buf, v)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

// Float

func (e *Encoder) Float32(v float32) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeFloat32(e.buf, v)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

func (e *Encoder) Float64(v float64) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)
	e.buf = EncodeFloat64(e.buf, v)
	end := len(e.buf)

	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

// Bytes/string

func (e *Encoder) Bytes(v []byte) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)

	var err error
	e.buf, err = EncodeBytes(e.buf, v)
	if err != nil {
		return e.fail(err)
	}

	end := len(e.buf)
	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

func (e *Encoder) String(v string) error {
	if e.err != nil {
		return e.err
	}

	start := len(e.buf)

	var err error
	e.buf, err = EncodeString(e.buf, v)
	if err != nil {
		return e.fail(err)
	}

	end := len(e.buf)
	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

// List

func (e *Encoder) BeginList() error {
	if e.err != nil {
		return e.err
	}

	// push list
	start := len(e.buf)
	tableStart := e.elements.offset()

	e.objects.pushList(start, tableStart)
	return nil
}

func (e *Encoder) Element() error {
	if e.err != nil {
		return e.err
	}

	// check list
	list, ok := e.objects.peek()
	switch {
	case !ok:
		return e.fail(errors.New("element: cannot encode element, not list encoder"))
	case list.type_ != objectTypeList:
		return e.fail(errors.New("element: cannot encode element, not list encoder"))
	}

	// pop data
	data := e.popData()

	// append element relative offset
	offset := uint32(data.end - list.start)
	element := listElement{offset: offset}
	e.elements.push(element)
	return nil
}

func (e *Encoder) EndList() error {
	if e.err != nil {
		return e.err
	}

	// pop list
	list, ok := e.objects.pop()
	switch {
	case !ok:
		return e.fail(errors.New("end list: not list encoder"))
	case list.type_ != objectTypeList:
		return e.fail(errors.New("end list: not list encoder"))
	}

	bsize := len(e.buf) - list.start
	table := e.elements.pop(list.tableStart)

	// encode list
	var err error
	e.buf, err = encodeListMeta(e.buf, bsize, table)
	if err != nil {
		return e.fail(err)
	}

	// push data entry
	start := list.start
	end := len(e.buf)
	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

// Message

func (e *Encoder) BeginMessage() error {
	if e.err != nil {
		return e.err
	}

	// push message
	start := len(e.buf)
	tableStart := e.fields.offset()

	e.objects.pushMessage(start, tableStart)
	return nil
}

func (e *Encoder) Field(tag uint16) error {
	if e.err != nil {
		return e.err
	}

	// check message
	message, ok := e.objects.peek()
	switch {
	case !ok:
		return e.fail(errors.New("field: cannot encode field, not message encoder"))
	case message.type_ != objectTypeMessage:
		return e.fail(errors.New("field: cannot encode field, not message encoder"))
	}

	// pop data
	data := e.popData()

	// insert field tag and relative offset
	f := messageField{
		tag:    tag,
		offset: uint32(data.end - message.start),
	}
	e.fields.insert(message.tableStart, f)
	return nil
}

func (e *Encoder) EndMessage() error {
	if e.err != nil {
		return e.err
	}

	// pop message
	message, ok := e.objects.pop()
	switch {
	case !ok:
		return e.fail(errors.New("end message: not message encoder"))
	case message.type_ != objectTypeMessage:
		return e.fail(errors.New("end message: not message encoder"))
	}

	bsize := len(e.buf) - message.start
	table := e.fields.pop(message.tableStart)

	// encode message
	var err error
	e.buf, err = encodeMessageMeta(e.buf, bsize, table)
	if err != nil {
		return e.fail(err)
	}

	// push data
	start := message.start
	end := len(e.buf)
	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

// Struct

func (e *Encoder) BeginStruct() error {
	if e.err != nil {
		return e.err
	}

	// push struct
	start := len(e.buf)
	e.objects.pushStruct(start)
	return nil
}

func (e *Encoder) StructField() error {
	if e.err != nil {
		return e.err
	}

	// check struct
	obj, ok := e.objects.peek()
	switch {
	case !ok:
		return e.fail(errors.New("field: cannot encode struct field, not struct encoder"))
	case obj.type_ != objectTypeStruct:
		return e.fail(errors.New("field: cannot encode struct field, not struct encoder"))
	}

	// just consume data
	e.popData()
	return nil
}

func (e *Encoder) EndStruct() error {
	if e.err != nil {
		return e.err
	}

	// pop struct
	obj, ok := e.objects.pop()
	switch {
	case !ok:
		return e.fail(errors.New("end struct: not struct encoder"))
	case obj.type_ != objectTypeStruct:
		return e.fail(errors.New("end struct: not struct encoder"))
	}

	bsize := len(e.buf) - obj.start

	// encode struct
	var err error
	e.buf, err = encodeStruct(e.buf, bsize)
	if err != nil {
		return e.fail(err)
	}

	// push data
	start := obj.start
	end := len(e.buf)
	if err := e.setData(start, end); err != nil {
		return e.fail(err)
	}
	return nil
}

// private

func (e *Encoder) fail(err error) error {
	if e.err != nil {
		return err
	}

	e.err = err
	return err
}

// data

// encodeData holds the last written data start/end.
// there is no data stack because the data must be consumed immediatelly after it is written.
type encodeData struct {
	start int
	end   int
}

func (e *Encoder) setData(start, end int) error {
	if e.data.start != 0 || e.data.end != 0 {
		return errors.New("encode: cannot encode more data, element/field must be written first")
	}

	e.data = encodeData{
		start: start,
		end:   end,
	}
	return nil
}

func (e *Encoder) popData() encodeData {
	d := e.data
	e.data = encodeData{}
	return d
}
