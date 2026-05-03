package money

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseMinor(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int64
	}{
		{name: "whole amount", value: "10", want: 1000},
		{name: "one decimal place", value: "10.5", want: 1050},
		{name: "two decimal places", value: "10.50", want: 1050},
		{name: "one satang", value: "0.01", want: 1},
		{name: "trim spaces", value: " 123.45 ", want: 12345},
		{name: "zero", value: "0", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMinor(tt.value)
			if err != nil {
				t.Fatalf("ParseMinor() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseMinor() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParseMinorRejectsInvalidInput(t *testing.T) {
	tests := []string{
		"",
		" ",
		"-1.00",
		"+1.00",
		".50",
		"1.",
		"1.234",
		"1.2.3",
		"1,000.00",
		"abc",
		"92233720368547758.08",
	}

	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			_, err := ParseMinor(value)
			if !errors.Is(err, ErrInvalidAmount) {
				t.Fatalf("ParseMinor() error = %v, want %v", err, ErrInvalidAmount)
			}
		})
	}
}

func TestFormatMinor(t *testing.T) {
	tests := []struct {
		name        string
		amountMinor int64
		want        string
	}{
		{name: "zero", amountMinor: 0, want: "0.00"},
		{name: "one satang", amountMinor: 1, want: "0.01"},
		{name: "whole amount", amountMinor: 1000, want: "10.00"},
		{name: "mixed amount", amountMinor: 12345, want: "123.45"},
		{name: "negative amount", amountMinor: -1050, want: "-10.50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatMinor(tt.amountMinor); got != tt.want {
				t.Fatalf("FormatMinor() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidatePositive(t *testing.T) {
	if err := ValidatePositive(1); err != nil {
		t.Fatalf("ValidatePositive() error = %v", err)
	}
	if err := ValidatePositive(0); !errors.Is(err, ErrAmountNotPositive) {
		t.Fatalf("ValidatePositive() error = %v, want %v", err, ErrAmountNotPositive)
	}
	if err := ValidatePositive(-1); !errors.Is(err, ErrAmountNotPositive) {
		t.Fatalf("ValidatePositive() error = %v, want %v", err, ErrAmountNotPositive)
	}
}

func TestSplitEqual(t *testing.T) {
	tests := []struct {
		name         string
		totalMinor   int64
		participants int
		want         []int64
	}{
		{name: "even split", totalMinor: 900, participants: 3, want: []int64{300, 300, 300}},
		{name: "remainder goes to last shares", totalMinor: 10000, participants: 3, want: []int64{3333, 3333, 3334}},
		{name: "two remainder units", totalMinor: 10001, participants: 3, want: []int64{3333, 3334, 3334}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitEqual(tt.totalMinor, tt.participants)
			if err != nil {
				t.Fatalf("SplitEqual() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SplitEqual() = %v, want %v", got, tt.want)
			}
			assertShareTotal(t, got, tt.totalMinor)
		})
	}
}

func TestSplitEqualRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name         string
		totalMinor   int64
		participants int
	}{
		{name: "zero total", totalMinor: 0, participants: 1},
		{name: "zero participants", totalMinor: 100, participants: 0},
		{name: "less than one minor unit per participant", totalMinor: 1, participants: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := SplitEqual(tt.totalMinor, tt.participants)
			if !errors.Is(err, ErrInvalidSplit) {
				t.Fatalf("SplitEqual() error = %v, want %v", err, ErrInvalidSplit)
			}
		})
	}
}

func TestValidateManualSplit(t *testing.T) {
	if err := ValidateManualSplit(1000, []int64{250, 750}); err != nil {
		t.Fatalf("ValidateManualSplit() error = %v", err)
	}
	if err := ValidateManualSplit(1000, []int64{250, 700}); !errors.Is(err, ErrSplitMismatch) {
		t.Fatalf("ValidateManualSplit() error = %v, want %v", err, ErrSplitMismatch)
	}
	if err := ValidateManualSplit(1000, []int64{250, 0, 750}); !errors.Is(err, ErrInvalidSplit) {
		t.Fatalf("ValidateManualSplit() error = %v, want %v", err, ErrInvalidSplit)
	}
	if err := ValidateManualSplit(0, []int64{0}); !errors.Is(err, ErrInvalidSplit) {
		t.Fatalf("ValidateManualSplit() error = %v, want %v", err, ErrInvalidSplit)
	}
}

func assertShareTotal(t *testing.T, shares []int64, want int64) {
	t.Helper()

	var got int64
	for _, share := range shares {
		got += share
	}
	if got != want {
		t.Fatalf("share total = %d, want %d", got, want)
	}
}
