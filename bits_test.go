// SPDX-FileCopyrightText: 2025 David Hendricks <david.hendricks@gmail.com>
//
// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"encoding/binary"
	"strings"
	"testing"
)

// TestPad verifies that pad() inserts the zero-extension where it belongs
// for both endiannesses and a range of integer sizes.
//
// Big-endian: the first character of the string is the MSB, so the
// partial (last) integer is zero-extended on its MSB side, i.e. zeros are
// inserted in front of it.
//
// Little-endian: the first byte of the string is the LSB. A trailing
// incomplete byte is completed with one zero inserted before the last
// character, and any remaining zero bytes are appended on the MSB side
// (end of string).
func TestPad(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		intsize  int
		order    binary.ByteOrder
		expected string
	}{
		// Big-endian.
		{"be: pad 2-byte int to 4 bytes", "aabb", 4, binary.BigEndian, "0000aabb"},
		{"be: pad 1-byte int to 8 bytes", "ff", 8, binary.BigEndian, "00000000000000ff"},
		{"be: pad 4-byte int to 8 bytes", "abcd", 8, binary.BigEndian, "000000000000abcd"},
		{"be: pad to 16 bytes", "aabb", 16, binary.BigEndian, strings.Repeat("0", 28) + "aabb"},
		{"be: odd-length single int", "abc", 2, binary.BigEndian, "0abc"},
		{"be: single nibble int", "c", 2, binary.BigEndian, "000c"},
		{"be: trailing partial after full ints", "aabbccdd" + "ee", 2, binary.BigEndian, "aabbccdd00ee"},
		{"be: trailing even partial after full int", "12345678" + "9a", 2, binary.BigEndian, "12345678009a"},
		{"be: intsize 1, trailing nibble", "aabbc", 1, binary.BigEndian, "aabb0c"},
		{"be: exact multiple unchanged", "aabbccdd", 2, binary.BigEndian, "aabbccdd"},
		{"be: 8-byte exact multiple unchanged", "1122334455667788", 8, binary.BigEndian, "1122334455667788"},

		// Little-endian.
		{"le: pad 4-byte int to 8 bytes", "abcd", 8, binary.LittleEndian, "abcd" + strings.Repeat("0", 12)},
		{"le: pad 1-byte int to 8 bytes", "ff", 8, binary.LittleEndian, "ff" + strings.Repeat("0", 14)},
		{"le: pad to 16 bytes", "aabb", 16, binary.LittleEndian, "aabb" + strings.Repeat("0", 28)},
		{"le: even trailing partial", "123456", 4, binary.LittleEndian, "12345600"},
		{"le: even trailing partial 2", "abcdef", 4, binary.LittleEndian, "abcdef00"},
		{"le: even trailing partial after full int", "123456789a", 2, binary.LittleEndian, "123456789a00"},
		{"le: intsize 1, trailing nibble", "aabbc", 1, binary.LittleEndian, "aabb0c"},
		{"le: odd trailing partial, one pad char", "abc", 2, binary.LittleEndian, "ab0c"},
		{"le: single nibble", "a", 1, binary.LittleEndian, "0a"},
		{"le: odd trailing partial after full int", "12345", 2, binary.LittleEndian, "12340500"},
		{"le: odd trailing partial after full int 2", "aabbc", 2, binary.LittleEndian, "aabb0c00"},
		{"le: odd trailing partial, 3 pad chars", "abcde", 4, binary.LittleEndian, "abcd0e00"},
		{"le: exact multiple unchanged", "aabbccdd", 2, binary.LittleEndian, "aabbccdd"},

		// Edge cases.
		{"empty string unchanged", "", 2, binary.BigEndian, ""},
		{"empty string unchanged le", "", 4, binary.LittleEndian, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pad(tt.str, tt.intsize, tt.order); got != tt.expected {
				t.Errorf("pad(%q, %d, %v) = %q, want %q", tt.str, tt.intsize, tt.order, got, tt.expected)
			}
		})
	}
}

func TestParseHex(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{"single arg", []string{"aabb"}, "aabb", false},
		{"multiple args joined", []string{"aa", "bb"}, "aabb", false},
		{"0x prefix stripped", []string{"0xaabb"}, "aabb", false},
		{"0X prefix stripped", []string{"0XAABB"}, "AABB", false},
		{"multiple prefixed args", []string{"0x12", "0x34"}, "1234", false},
		{"bare 0x rejected", []string{"0x"}, "", true},
		{"whitespace stripped", []string{"aabb  ", "\tcc\ndd"}, "aabbccdd", false},
		{"uppercase kept", []string{"AB"}, "AB", false},
		{"invalid char", []string{"12g4"}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseHex(tt.args)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseHex(%v) error = %v, wantErr %v", tt.args, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseHex(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		high    int
		low     int
		wantErr bool
	}{
		{"valid", "63:0", 63, 0, false},
		{"single bit", "5:5", 5, 5, false},
		{"negative low parses (caller bound-checks)", "8:-1", 8, -1, false},
		{"missing colon", "5", 0, 0, true},
		{"too many colons", "5:4:3", 0, 0, true},
		{"bad high", "abc:0", 0, 0, true},
		{"bad low", "5:xyz", 5, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			high, low, err := parseRange(tt.s)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRange(%q) error = %v, wantErr %v", tt.s, err, tt.wantErr)
			}
			if err == nil && (high != tt.high || low != tt.low) {
				t.Errorf("parseRange(%q) = %d:%d, want %d:%d", tt.s, high, low, tt.high, tt.low)
			}
		})
	}
}

func TestParseEndianness(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    binary.ByteOrder
		wantErr bool
	}{
		{"default is BE", "", binary.BigEndian, false},
		{"little-endian", "l", binary.LittleEndian, false},
		{"big-endian", "b", binary.BigEndian, false},
		{"unknown", "x", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _, err := parseEndianness(tt.s)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseEndianness(%q) error = %v, wantErr %v", tt.s, err, tt.wantErr)
			}
			if err == nil && got != tt.want {
				t.Errorf("parseEndianness(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestByteAt(t *testing.T) {
	b := []byte{0x12, 0x34, 0x56, 0x78}
	tests := []struct {
		name  string
		order binary.ByteOrder
		pos   int
		want  byte
	}{
		{"be: pos is MSB", binary.BigEndian, 3, 0x12},
		{"be: pos is LSB", binary.BigEndian, 0, 0x78},
		{"le: pos is LSB", binary.LittleEndian, 0, 0x12},
		{"le: pos is MSB", binary.LittleEndian, 3, 0x78},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := byteAt(tt.order, b, tt.pos); got != tt.want {
				t.Errorf("byteAt(%v, %x, %d) = %#x, want %#x", tt.order, b, tt.pos, got, tt.want)
			}
		})
	}
}
