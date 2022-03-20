package spec

import (
	"encoding/binary"
	"math"
)

const (
	messageFieldSmallSize = 1 + 2 // tag(1) + offset(2)
	messageFieldBigSize   = 2 + 4 // tag(2) + offset(4)
)

type messageMeta struct {
	table messageTable

	data uint32 // message data size
	big  bool   // big/small table format
}

// count returns the number of fields in the message.
func (m messageMeta) count() int {
	return m.table.count(m.big)
}

// offset returns field end offset by a tag or -1.
func (m messageMeta) offset(tag uint16) int {
	if m.big {
		return m.table.offset_big(tag)
	} else {
		return m.table.offset_small(tag)
	}
}

// offsetByIndex returns field end offset by an index or -1.
func (m messageMeta) offsetByIndex(i int) int {
	if m.big {
		return m.table.offsetByIndex_big(i)
	} else {
		return m.table.offsetByIndex_small(i)
	}
}

// field returns a field by an index or false,
func (m messageMeta) field(i int) (messageField, bool) {
	if m.big {
		return m.table.field_big(i)
	} else {
		return m.table.field_small(i)
	}
}

// messageTable is a serialized array of message fields ordered by tags.
// the serialization format depends on whether the message is big or small, see isBigMessage().
//
//          field0                field1                field2
//  +---------------------+---------------------+---------------------+
// 	|  tag0 |   offset0   |  tag1 |   offset1   |  tag2 |   offset3   |
//  +---------------------+---------------------+---------------------+
//
type messageTable []byte

// messageField specifies a tag and a value offset in a message byte array.
//
//  +----------+-------------------+
// 	| tag(1/2) |    offset(2/4)    |
//  +----------+-------------------+
//
type messageField struct {
	tag    uint16
	offset uint32
}

// isBigList returns true if any field tag > uint8 or offset > uint16.
func isBigMessage(table []messageField) bool {
	ln := len(table)
	if ln == 0 {
		return false
	}

	for i := ln - 1; i >= 0; i-- {
		field := table[i]

		switch {
		case field.tag > math.MaxUint8:
			return true
		case field.offset > math.MaxUint16:
			return true
		}
	}

	return false
}

// count returns the number of fields in the table.
func (t messageTable) count(big bool) int {
	var size int
	if big {
		size = messageFieldBigSize
	} else {
		size = messageFieldSmallSize
	}
	return len(t) / size
}

// offset

func (t messageTable) offset_big(tag uint16) int {
	size := messageFieldBigSize
	n := len(t) / size

	// binary search table
	left, right := 0, (n - 1)
	for left <= right {
		// middle
		middle := int(uint(left+right) >> 1) // avoid overflow

		// offset
		off := middle * size
		b := t[off : off+size]

		// current tag
		cur := binary.BigEndian.Uint16(b)

		// check current
		switch {
		case cur < tag:
			left = middle + 1

		case cur > tag:
			right = middle - 1

		case cur == tag:
			// read offset after tag
			return int(binary.BigEndian.Uint32(b[2:]))
		}
	}

	return -1
}

func (t messageTable) offset_small(tag uint16) int {
	size := messageFieldSmallSize
	n := len(t) / size

	// binary search table
	left, right := 0, (n - 1)
	for left <= right {
		// middle
		middle := int(uint(left+right) >> 1) // avoid overflow

		// offset
		off := middle * size
		b := t[off : off+size]

		// current tag
		cur := uint16(b[0])

		// check current
		switch {
		case cur < tag:
			left = middle + 1

		case cur > tag:
			right = middle - 1

		case cur == tag:
			// read offset after tag
			return int(binary.BigEndian.Uint16(b[1:]))
		}
	}

	return -1
}

// offsetByIndex

func (t messageTable) offsetByIndex_big(i int) int {
	size := messageFieldBigSize
	n := len(t) / size

	// check count
	switch {
	case i < 0:
		return -1
	case i >= n:
		return -1
	}

	// field offset
	off := i * size

	// read end after tag
	return int(binary.BigEndian.Uint32(t[off+2:]))
}

func (t messageTable) offsetByIndex_small(i int) int {
	size := messageFieldSmallSize
	n := len(t) / size

	// check count
	switch {
	case i < 0:
		return -1
	case i >= n:
		return -1
	}

	// field offset
	off := i * size

	// read end after tag
	return int(binary.BigEndian.Uint16(t[off+1:]))
}

// field

func (t messageTable) field_big(i int) (f messageField, ok bool) {
	size := messageFieldBigSize
	n := len(t) / size

	// check count
	switch {
	case i < 0:
		return
	case i >= n:
		return
	}

	off := i * size
	b := t[off : off+size]

	f = messageField{
		tag:    binary.BigEndian.Uint16(b),
		offset: binary.BigEndian.Uint32(b[2:]),
	}

	ok = true
	return
}

func (t messageTable) field_small(i int) (f messageField, ok bool) {
	size := messageFieldSmallSize
	n := len(t) / size

	// check count
	switch {
	case i < 0:
		return
	case i >= n:
		return
	}

	off := i * size
	b := t[off : off+size]

	f = messageField{
		tag:    uint16(b[0]),
		offset: uint32(binary.BigEndian.Uint16(b[1:])),
	}

	ok = true
	return
}

// fields parses the table and returns a slice of fields.
func (t messageTable) fields(big bool) []messageField {
	n := t.count(big)

	result := make([]messageField, 0, n)
	for i := 0; i < n; i++ {
		var field messageField
		var ok bool

		if big {
			field, ok = t.field_big(i)
		} else {
			field, ok = t.field_small(i)
		}
		if !ok {
			continue
		}

		result = append(result, field)
	}
	return result
}
