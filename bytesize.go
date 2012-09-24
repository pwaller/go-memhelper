package memhelper

import (
	"fmt"
)

type ByteSize int64

const (
	_            = iota
	kiB ByteSize = 1 << (10 * iota)
	MiB
	GiB
	TiB
)

const (
	_           = iota
	kB ByteSize = 1000
	MB          = 1000 * kB
	GB          = 1000 * MB
	TB          = 1000 * GB
)

func (b ByteSize) UnitDivisor(decimal bool) (string, ByteSize) {
	switch decimal {
	case true:
		switch {
		case b >= TB:
			return "TB", TB
		case b >= GB:
			return "GB", GB
		case b >= MB:
			return "MB", MB
		case b >= kB:
			return "kB", kB
		}
	case false:
		switch {
		case b/GiB > 1000:
			return "TiB", TiB
		case b/MiB > 1000:
			return "GiB", GiB
		case b/kiB > 1000:
			return "MiB", MiB
		case b > 1000:
			return "kiB", kiB
		}
	}
	return "B", ByteSize(1)
}

func (b ByteSize) String() string {
	unit, divisor := b.UnitDivisor(true)
	return fmt.Sprintf("%.2f%s", float64(b)/float64(divisor), unit)
}

// TODO: Implement left/right align
func (b ByteSize) Format(f fmt.State, c rune) {
	var decimal bool
	switch c {
	case 'd':
		decimal = true
	case 'b':
		decimal = false
	case 'v':
		fmt.Fprintf(f, "%s", b.String())
		return
	default:
		fmt.Fprintf(f, "%%!%c(ByteSize=%s)", c, b.String())
		return
	}

	unit, divisor := b.UnitDivisor(decimal)
	fmtstring := "%"

	if w, ok := f.Width(); ok {
		fmtstring += fmt.Sprintf("%d", w-len(unit)-1)
	}
	if p, ok := f.Precision(); ok {
		fmtstring += fmt.Sprintf(".%d", p)
	}
	fmtstring += "f %s"

	fmt.Fprintf(f, fmtstring, float64(b)/float64(divisor), unit)
}
