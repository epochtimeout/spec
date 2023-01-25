package spec

import (
	"math/rand"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestEncoderState_Size__should_be_lte_1kb(t *testing.T) {
	s := unsafe.Sizeof(encoderState{})
	if s > 1024 {
		t.Fatal(s)
	}
}

// list buffer

func TestListBuffer_push__should_append_element_to_last_list(t *testing.T) {
	matrix := [][]listElement{
		testListElementsN(1),
		testListElementsN(10),
		testListElementsN(100),
		testListElementsN(10),
		testListElementsN(1),
		testListElementsN(0),
		testListElementsN(3),
	}

	buffer := listStack{}
	offsets := []int{}

	// build buffer
	for _, elements := range matrix {
		offset := buffer.offset()
		offsets = append(offsets, offset)

		// push
		for _, elem := range elements {
			buffer.push(elem)
		}
	}

	// check buffer
	for i := len(offsets) - 1; i >= 0; i-- {
		offset := offsets[i]

		// pop table
		ff := buffer.pop(offset)
		elements := matrix[i]

		// check table
		require.Equal(t, elements, ff)
	}
}

// message buffer

func TestMessageBuffer_Insert__should_insert_field_into_table_ordered_by_tags(t *testing.T) {
	matrix := [][]messageField{
		testMessageFieldsN(1),
		testMessageFieldsN(10),
		testMessageFieldsN(100),
		testMessageFieldsN(10),
		testMessageFieldsN(1),
		testMessageFieldsN(0),
		testMessageFieldsN(3),
	}

	buffer := messageStack{}
	offsets := []int{}

	// build buffer
	for _, fields := range matrix {
		offset := buffer.offset()
		offsets = append(offsets, offset)

		// copy
		ff := make([]messageField, len(fields))
		copy(ff, fields)

		// shuffle
		rand.Shuffle(len(ff), func(i, j int) {
			ff[j], ff[i] = ff[i], ff[j]
		})

		// insert
		for _, f := range ff {
			buffer.insert(offset, f)
		}
	}

	// check buffer
	for i := len(offsets) - 1; i >= 0; i-- {
		offset := offsets[i]

		// pop table
		ff := buffer.pop(offset)
		fields := matrix[i]

		// check table
		require.Equal(t, fields, ff)
	}
}
