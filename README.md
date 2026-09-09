# bits

Print the bits of hexadecimal integers.

`bits` takes one or more hexadecimal numbers, groups them into integers, and
prints each bit. Non-flag arguments are processed as a byte array rather than
converted to uint, thus you are not limited to your host machine's uint size
with this tool and can print bits from arbitrary-length integers.

This is useful for analyzing hardware register values such as raw hex coming
out of `lspci -x` or `ipmitool`, protocol fields, or any other bit-packed data.
It fits somewhere between a calculator and a specialized/context-aware decoder.

## Building

``` sh
go build -o bits .
```

or install directly:

``` sh
go install github.com/dhendrix/bits@latest
```

## Examples

Print bits in a given range:

``` console
$ bits.go -s 2 aa bb -r 11:4
BE 0xaabb:
bit[11]: 1
bit[10]: 0
bit[ 9]: 1
bit[ 8]: 0
bit[ 7]: 1
bit[ 6]: 0
bit[ 5]: 1
bit[ 4]: 1
```

Print the value of a range you care about:
``` console
$ bits.go -s 2 ab cd -r 11:4 -v
BE 0xabcd[11:4]: 0xbc
```

Print value from raw hex, e.g. from `lspci -x`:
```
$ bits.go -s 2 -v -a -el 86 80 0d 46 
LE 0x8680[15:0]: 0x8086
LE 0x0d46[15:0]: 0x460d
```

Extract byte values from a 32-bit integer:
``` console
$ bits -r 7:0 -v 11223344
BE 0x0000000011223344[7:0]: 0x44

$ bits -r 15:8 -v 11223344
BE 0x0000000011223344[15:8]: 0x33

$ bits -a -r 15:0 -v 11223344
BE 0x0000000011223344[15:0]: 0x3344
```

`bits` will automatically zero-extend integers when necessary:
``` console
$ bits.go -s 1 0a b
BE 0x0a:
bit[ 3]: 1
bit[ 2]: 0
bit[ 1]: 1
bit[ 0]: 0

BE 0x0b:
bit[ 3]: 1
bit[ 2]: 0
bit[ 1]: 1
bit[ 0]: 1
```

## Options
```
  -a, --all                Print all bits, including leading zeroes (suppressed by default)

  -e, --endianness l\|b    Endianness of input: `l/little` = LE, b/big = BE (default)

  -r, --range high:low     Inclusive range of bits to print; bit 0 is the least significant

  -s, --size <n>           Size of an integer in bytes (default: size of an int)

  -v, --value              Print hex value instead of individual bits (use with -r)

  -h, --help               Show flag help
```

## Tests

``` console
$ go test .
```

## License

BSD-3-Clause; see the copyright headers in the source files.
