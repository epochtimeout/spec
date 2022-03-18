package spec

import (
	"fmt"
	"testing"
)

func _BenchmarkSizeDistribution(b *testing.B) {
	msg := newTestObject()

	w := NewWriter()
	mw := BeginTestMessage(w)
	if err := msg.Write(mw); err != nil {
		b.Fatal(err)
	}
	data, err := w.End()
	if err != nil {
		b.Fatal(err)
	}

	_, p, err := computeSizeDistribution(data)
	if err != nil {
		b.Fatal(err)
	}
	if p.size == 0 {
		b.Fatal()
	}

	b.Fatalf("%+v", p)
}

type sizeDistrib struct {
	// total size
	size int

	// total distribution
	meta   int
	tables int
	data   int

	// meta distribution
	types int
	sizes int

	// data distribution
	values  int
	bytes   int
	strings int
}

type sizeDistribPercent struct {
	// total size
	size int

	// total distribution
	meta   float32
	tables float32
	data   float32

	// meta distribution
	types float32
	sizes float32

	// data distribution
	values  float32
	bytes   float32
	strings float32
}

func computeSizeDistribution(b []byte) (*sizeDistrib, *sizeDistribPercent, error) {
	d := &sizeDistrib{}
	p := &sizeDistribPercent{}
	if len(b) == 0 {
		return d, p, nil
	}

	if err := _computeSizeDistribution(b, d); err != nil {
		return nil, nil, err
	}
	d.size = len(b)
	d.meta = d.types + d.sizes
	d.data = d.values + d.bytes + d.strings

	p.size = d.size
	p.meta = float32(d.meta) / float32(d.size)
	p.tables = float32(d.tables) / float32(d.size)
	p.data = float32(d.data) / float32(d.size)

	p.types = float32(d.types) / float32(d.size)
	p.sizes = float32(d.sizes) / float32(d.size)

	p.values = float32(d.values) / float32(d.size)
	p.bytes = float32(d.bytes) / float32(d.size)
	p.strings = float32(d.strings) / float32(d.size)
	return d, p, nil
}

func _computeSizeDistribution(b []byte, d *sizeDistrib) error {
	t, n := decodeType(b)
	if n < 0 {
		return fmt.Errorf("invalid type")
	}
	d.types += n

	switch t {
	case TypeNil, TypeTrue, TypeFalse:
		return nil

	case TypeByte, TypeInt32, TypeInt64:
		_, vn := decodeInt64(b)
		d.values += vn - n

	case TypeUint32, TypeUint64:
		_, vn := decodeInt64(b)
		d.values += vn - n

	case TypeFloat32, TypeFloat64:
		_, vn := decodeFloat64(b)
		d.values += vn - n

	case TypeBytes:
		off := len(b) - 1
		size, sn := decodeBytesSize(b[:off])
		if sn < 0 {
			return fmt.Errorf("invalid bytes size")
		}

		d.sizes += sn
		d.bytes += int(size)

	case TypeString:
		off := len(b) - 1
		size, sn := decodeStringSize(b[:off])
		if n < 0 {
			return fmt.Errorf("invalid string size")
		}

		d.sizes += sn
		d.strings += int(size)

	case TypeList, TypeListBig:
		// read table size
		off := len(b) - 1
		tsize, tn := decodeListTableSize(b[:off])
		if tn < 0 {
			return fmt.Errorf("invalid list table size")
		}

		// read data size
		off -= tn
		_, dn := decodeListBodySize(b[:off])
		if dn < 0 {
			return fmt.Errorf("invalid list data size")
		}

		d.sizes += tn + dn
		d.tables += int(tsize)

		// read list
		list, _, err := DecodeList(b, ReadValue)
		if err != nil {
			return err
		}

		// read elements
		for i := 0; i < list.Count(); i++ {
			elem := list.ElementBytes(i)
			if err := _computeSizeDistribution(elem, d); err != nil {
				return err
			}
		}

	case TypeMessage, TypeMessageBig:
		// read table size
		off := len(b) - 1
		tsize, tn := decodeMessageTableSize(b[:off])
		if tn < 0 {
			return fmt.Errorf("invalid message table size")
		}

		// read data size
		off -= tn
		_, dn := decodeMessageBodySize(b[:off])
		if dn < 0 {
			return fmt.Errorf("invalid message data size")
		}

		d.sizes += tn + dn
		d.tables += int(tsize)

		// read message meta
		msg, _, err := DecodeMessage(b)
		if err != nil {
			return err
		}

		// read fields
		for i := 0; i < msg.Count(); i++ {
			field := msg.FieldByIndex(i)
			if err := _computeSizeDistribution(field, d); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("unsupported type %d", t)
	}

	return nil
}
