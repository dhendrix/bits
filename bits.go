// SPDX-FileCopyrightText: 2025 David Hendricks <david.hendricks@gmail.com>
//
// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"

	flag "github.com/spf13/pflag"
)

var (
	allFlag    = flag.BoolP("all", "a", false, "print all bits (including leading zeroes)")
	valueFlag  = flag.BoolP("value", "v", false, "print value (in hex) of bits in given range")
	rangeFlag  = flag.StringP("range", "r", "", "range in the form high:low")
	endianFlag = flag.StringP("endianness", "e", "", "endianness of input (l = LE, b = BE")
	sizeFlag   = flag.IntP("size", "s", strconv.IntSize/8, "size of an integer (bytes)")
)

func printHelp() {
	fmt.Println("Usage: bits <args> [-r high:low] <num>...")
	fmt.Println("Optional arguments:")
	fmt.Println("    -a/--all           Print all bits, including leading zeroes")
	fmt.Println("    -e/--endianness    Endianness of input (l = LE, b = BE")
	//	fmt.Println("    -f/--format    Format of input")
	fmt.Println("    -r/--range         Range of bits to print")
	fmt.Println("    -s/--size          Size of an integer (bytes)")
	fmt.Println("    -v/--value         Print value (little-endian) of bits in given range (-r required)")
}

func parseEndianness(s string) (binary.ByteOrder, string, error) {
	switch s {
	case "":
		// assume user is writing values from MSB on the left to LSB on the right
		return binary.BigEndian, "BE", nil
	case "l":
		return binary.LittleEndian, "LE", nil
	case "b":
		return binary.BigEndian, "BE", nil
	default:
		return nil, "", fmt.Errorf("unknown endianness specified: %v", s)
	}
}

// parseHex joins the arguments into a single hex string. A "0x" prefix is
// stripped from each argument, whitespace is discarded, and any other
// character is an error.
//
// The result is then grouped by intSize (split or joined) and padded where
// needed. A few examples:
//
//  1. bits -s 2 86 80 38 7a     is processed as "8680" and "387a"
//  2. bits -s 2 -e l 86 80 38 7a is processed as "8086" and "7a38"
//  3. bits -s 1 aa bb c         is processed as "aa" "bb" "0c"
//  4. bits -s 4 -e b aabb       is processed as "0000aabb"
func parseHex(args []string) (string, error) {
	var sb strings.Builder
	for _, arg := range args {
		if len(arg) > 2 && (strings.HasPrefix(arg, "0x") || strings.HasPrefix(arg, "0X")) {
			arg = arg[2:]
		}
		for _, c := range arg {
			switch {
			case (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F'):
				sb.WriteRune(c)
			case unicode.IsSpace(c):
				// Whitespace is stripped.
			default:
				return "", fmt.Errorf("invalid character: %c", c)
			}
		}
	}
	return sb.String(), nil
}

// pad zero-extends str to the nearest intSize boundary. This helps with
// printing leading zeroes (if desired).
//
// For big-endian input the first character is the MSB, so zeros are
// inserted in front of the trailing partial integer.
//
// For little-endian input the first byte is the LSB: a trailing incomplete
// byte is completed with one zero in its high-nibble position, and any
// remaining zero bytes are appended on the MSB side (end of string).
func pad(str string, intSize int, order binary.ByteOrder) string {
	if len(str)%(intSize*2) == 0 {
		return str
	}

	lastIntPos := (len(str) / (intSize * 2)) * (intSize * 2)
	zeros := strings.Repeat("0", lastIntPos+intSize*2-len(str))

	if order == binary.BigEndian {
		return str[:lastIntPos] + zeros + str[lastIntPos:]
	}

	if len(str)%2 != 0 {
		// The trailing byte is incomplete: only its high nibble is missing,
		// so a single zero goes in front of the last character.
		str = str[:len(str)-1] + "0" + str[len(str)-1:]
		zeros = zeros[1:]
	}
	return str + zeros
}

func parseRange(s string) (high, low int, err error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid -r argument, must be in form high:low")
	}

	high, err = strconv.Atoi(parts[0])
	if err != nil {
		return high, 0, fmt.Errorf("invalid high value for range %d:%d", high, 0)
	}

	low, err = strconv.Atoi(parts[1])
	if err != nil {
		return high, low, fmt.Errorf("invalid low value for range %d:%d", high, low)
	}

	return high, low, nil
}

type options struct {
	intSize int
	all     bool
	value   bool
	high    int
	low     int
}

// byteAt returns the byte at position pos counted from the least
// significant byte (pos 0 = LSB) for the given byte order.
func byteAt(order binary.ByteOrder, b []byte, pos int) byte {
	if order == binary.BigEndian {
		return b[len(b)-1-pos]
	}
	return b[pos]
}

// printBits prints the bits of a single integer, most significant byte
// first, or the value of the selected range when opts.value is set.
func printBits(order binary.ByteOrder, name, valStr string, opts *options) {
	// valStr is validated, even-length hex, so decoding cannot fail.
	bytes, _ := hex.DecodeString(valStr)

	if !opts.value {
		fmt.Printf("%s 0x%s:\n", name, valStr)
	}

	var leadingZero = true // assume the most significant bit is a leading zero
	var value uint

	maxBit := opts.high
	if opts.all {
		maxBit = opts.intSize*8 - 1
	}
	width := len(strconv.Itoa(maxBit))

	for pos := opts.intSize - 1; pos >= 0; pos-- { // iterate over bytes, MSB first
		b := byteAt(order, bytes, pos)
		for bit := 7; bit >= 0; bit-- { // iterate over bits in each byte
			bitPos := pos*8 + bit
			bitVal := (uint(b) >> bit) & 1

			if !opts.all {
				if bitPos > opts.high {
					continue
				} else if bitPos < opts.low {
					break
				}

				// Suppress leading zeroes.
				if leadingZero {
					if bitVal == 0 {
						continue
					}
					leadingZero = false
				}
			}

			if opts.value {
				value |= bitVal << bitPos
			} else {
				fmt.Printf("bit[%*d]: %d\n", width, bitPos, bitVal)
			}
		}
	}

	if opts.value {
		var width int
		if opts.all {
			width = (opts.high - opts.low) + 4
			width -= width % 4
			width /= 4
		}

		var mask uint
		if opts.high == strconv.IntSize-1 {
			mask = ^uint(0)
		} else {
			mask = (1 << (opts.high + 1 - opts.low)) - 1
		}
		value = (value >> opts.low) & mask
		// TODO: add a flag to select output endianness?
		fmt.Printf("%s 0x%s[%d:%d]: 0x%0*x\n", name, valStr, opts.high, opts.low, width, value)
	}
}

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		printHelp()
		os.Exit(1)
	}

	if err := run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if *sizeFlag < 1 {
		return errors.New("integer size must be greater than 0")
	}
	intSize := *sizeFlag

	order, name, err := parseEndianness(*endianFlag)
	if err != nil {
		return err
	}

	hexStr, err := parseHex(args)
	if err != nil {
		return err
	}
	hexStr = pad(hexStr, intSize, order)

	var high, low int
	if *rangeFlag != "" {
		if high, low, err = parseRange(*rangeFlag); err != nil {
			return err
		}
		if high >= intSize*8 {
			return fmt.Errorf("invalid high value for range %d:%d", high, low)
		}
		if low < 0 {
			return fmt.Errorf("invalid low value for range %d:%d", high, low)
		}
		if *valueFlag && high >= strconv.IntSize {
			// TODO: need to rework the masking logic below to accommodate
			// higher ranges. Ideally this should accept any range of bits
			// that can fit in a uint, e.g. if high - low >= strconv.IntSize.
			return fmt.Errorf("cannot show values for range %d:%d", high, low)
		}
	} else {
		high = intSize*8 - 1
	}

	opts := &options{
		intSize: intSize,
		all:     *allFlag,
		value:   *valueFlag,
		high:    high,
		low:     low,
	}

	for i := 0; i < len(hexStr); i += intSize * 2 { // iterate over integers
		printBits(order, name, hexStr[i:i+intSize*2], opts)
		if !opts.value && i+intSize*2 < len(hexStr) {
			fmt.Println()
		}
	}

	return nil
}
