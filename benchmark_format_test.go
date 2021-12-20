package spec

import "testing"

func BenchmarkFieldTable_field(b *testing.B) {
	fields := testMessageFieldsN(100)
	data, size, err := _writeMessageTable(nil, fields)
	if err != nil {
		b.Fatal(err)
	}

	table, err := _readMessageTable(data, size)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	last := len(fields) - 1
	for i := 0; i < b.N; i++ {
		f, ok := table.field(last)
		if !ok || f.tag == 0 || f.offset == 0 {
			b.Fatal()
		}
	}
}

func BenchmarkFieldTable_offset(b *testing.B) {
	fields := testMessageFieldsN(100)
	data, size, err := _writeMessageTable(nil, fields)
	if err != nil {
		b.Fatal(err)
	}

	table, err := _readMessageTable(data, size)
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	last := len(fields) - 1
	tag := fields[last].tag
	for i := 0; i < b.N; i++ {
		off := table.offset(tag)
		if off < 0 {
			b.Fatal()
		}
	}
}
