package service

import "testing"

func TestValidLun_ValidNumbers(t *testing.T) {
	t.Parallel()

	tests := []string{
		"4539 1488 0343 6467",
		"4556-7375-8689-9855",
		"79927398713",
		"6011000990139424",
		"371449635398431",
		"378282246310005",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			if !ValidLun(tt) {
				t.Fatalf("expected valid Luhn number: %q", tt)
			}
		})
	}
}

func TestValidLun_InvalidChecksum(t *testing.T) {
	t.Parallel()

	tests := []string{
		"4539 1488 0343 6466",
		"79927398710",
		"371449635398432",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			if ValidLun(tt) {
				t.Fatalf("expected invalid Luhn checksum: %q", tt)
			}
		})
	}
}

func TestValidLun_InvalidCharacters(t *testing.T) {
	t.Parallel()

	tests := []string{
		"4539 1488 0343 64a7",
		"1234_5678",
		"１２３４５６",
		"abcd",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			if ValidLun(tt) {
				t.Fatalf("expected invalid input due to characters: %q", tt)
			}
		})
	}
}

func TestValidLun_OnlySeparators(t *testing.T) {
	t.Parallel()

	tests := []string{
		"",
		" ",
		"-",
		" - - ",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			if ValidLun(tt) {
				t.Fatalf("expected false for separators only: %q", tt)
			}
		})
	}
}

func TestValidLun_TooShort(t *testing.T) {
	t.Parallel()

	tests := []string{
		"0",
		"9",
		"1-",
		" 4 ",
	}

	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			if ValidLun(tt) {
				t.Fatalf("expected false for too short input: %q", tt)
			}
		})
	}
}

func TestValidLun_DoubleDigitReduction(t *testing.T) {
	t.Parallel()

	if !ValidLun("18") {
		t.Fatalf("expected valid Luhn for '18'")
	}
}
