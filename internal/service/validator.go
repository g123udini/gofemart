package service

import "unicode"

func ValidLun(s string) bool {
	sum := 0
	doubled := false
	digits := 0

	for i := len(s) - 1; i >= 0; i-- {
		r := rune(s[i])

		if r == ' ' || r == '-' {
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}

		d := int(r - '0')
		digits++

		if doubled {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
		doubled = !doubled
	}

	if digits < 2 {
		return false
	}

	return sum%10 == 0
}
