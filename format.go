package spec

import (
	"encoding/binary"
)

// listElement specifies an element value offset in list data array.
//
//  +-----------------+
// 	|    offset(4)    |
//  +-----------------+
//
type listElement struct {
	offset uint32
}

const listElementSize = 4

// listTable is a serialized array of list element offsets ordered by index.
//
//  +--------------+--------------+--------------+
// 	|    off0(4)   |    off1(4)   |    off2(4)   |
//  +--------------+--------------+--------------+
//
type listTable []byte

// count returns the number of elements in the table.
func (t listTable) count() int {
	return len(t) / listElementSize
}

// elements parses the table and returns a slice of elements
func (t listTable) elements() []listElement {
	n := t.count()

	result := make([]listElement, 0, n)
	for i := 0; i < n; i++ {
		off := t.offset(i)
		elem := listElement{uint32(off)}
		result = append(result, elem)
	}
	return result
}

// offset returns an element offset by its index or -1.
func (t listTable) offset(i int) int {
	n := t.count()
	switch {
	case i < 0:
		return -1
	case i >= n:
		return -1
	}

	off := i * listElementSize
	b := t[off : off+listElementSize]
	offset := binary.BigEndian.Uint32(b)
	return int(offset)
}

// listStack acts as a buffer for nested list elements.
//
// Each list externally stores its start offset in the buffer, and provides the offset
// when inserting new elements.
//
//	        list0              sublist1            sublist2
//	+-------------------+-------------------+-------------------+
//	| e0 | e1 | e2 | e3 | e0 | e1 | e2 | e3 | e0 | e1 | e2 | e3 |
//	+-------------------+-------------------+-------------------+
//
type listStack struct {
	stack []listElement
}

func (s *listStack) reset() {
	s.stack = s.stack[:0]
}

// offset returns the next list buffer offset.
func (s *listStack) offset() int {
	return len(s.stack)
}

// push appends a new element to the last list.
func (s *listStack) push(elem listElement) {
	s.stack = append(s.stack, elem)
}

// pop pops a list table starting at offset.
func (s *listStack) pop(offset int) []listElement {
	table := s.stack[offset:]
	s.stack = s.stack[:offset]
	return table
}

// messageField specifies a field tag and a field value offset in message data array.
//
//  +--------+-----------------+
// 	| tag(2) |    offset(4)    |
//  +--------+-----------------+
//
type messageField struct {
	tag    uint16
	offset uint32
}

const messageFieldSize = 2 + 4 // tag(2) + offset(4)

// messageTable is a serialized array of message fields ordered by tags.
//
//          field0                field1                field2
//  +---------------------+---------------------+---------------------+
// 	|  tag0 |   offset0   |  tag1 |   offset1   |  tag2 |   offset3   |
//  +---------------------+---------------------+---------------------+
//
type messageTable []byte

// count returns the number of fields in the table.
func (t messageTable) count() int {
	return len(t) / messageFieldSize
}

// field returns a field by its index or false,
func (t messageTable) field(i int) (f messageField, ok bool) {
	n := t.count()
	switch {
	case i < 0:
		return
	case i >= n:
		return
	}

	off := i * messageFieldSize
	b := t[off : off+messageFieldSize]

	f = messageField{
		tag:    binary.BigEndian.Uint16(b),
		offset: binary.BigEndian.Uint32(b[2:]),
	}
	ok = true
	return
}

// fields parses the table and returns a slice of fields.
func (t messageTable) fields() []messageField {
	n := t.count()

	result := make([]messageField, 0, n)
	for i := 0; i < n; i++ {
		field, ok := t.field(i)
		if !ok {
			continue
		}
		result = append(result, field)
	}
	return result
}

// offset finds and returns a field offset by its tag or -1.
func (t messageTable) offset(tag uint16) int {
	n := t.count()
	if n == 0 {
		return -1
	}

	// binary search table
	left, right := 0, (n - 1)
	for left <= right {
		// calc offset
		middle := int(uint(left+right) >> 1) // avoid overflow
		off := middle * messageFieldSize
		b := t[off : off+messageFieldSize]

		// read current tag
		cur := binary.BigEndian.Uint16(b)

		// check current
		switch {
		case cur < tag:
			left = middle + 1
		case cur > tag:
			right = middle - 1
		case cur == tag:
			offset := binary.BigEndian.Uint32(b[2:])
			return int(offset)
		}
	}
	return -1
}

// messageStack acts as a buffer for nested message fields.
//
// Each message externally stores its start offset in the buffer, and provides the offset
// when inserting new fields. Message fields are kept sorted by tags using the insertion sort.
//
//	       message0          submessage1         submessage2
//	+-------------------+-------------------+-------------------+
//	| f0 | f1 | f2 | f3 | f0 | f1 | f2 | f3 | f0 | f1 | f2 | f3 |
//	+-------------------+-------------------+-------------------+
//
type messageStack struct {
	stack []messageField
}

func (s *messageStack) reset() {
	s.stack = s.stack[:0]
}

// offset returns the next message table buffer offset.
func (s *messageStack) offset() int {
	return len(s.stack)
}

// insert inserts a new field into the last table starting at offset, keeps the table sorted.
func (s *messageStack) insert(offset int, f messageField) {
	// append new field
	s.stack = append(s.stack, f)

	// get table
	table := s.stack[offset:]

	// walk table in reverse order
	// move new field to its position
	// using insertion sort
	for i := len(table) - 1; i > 0; i-- {
		left := table[i-1]
		right := table[i]

		if left.tag < right.tag {
			// sorted
			break
		}

		// swap fields
		table[i-1] = right
		table[i] = left
	}
}

// pop pops a message table starting at offset.
func (s *messageStack) pop(offset int) []messageField {
	table := s.stack[offset:]
	s.stack = s.stack[:offset]
	return table
}
