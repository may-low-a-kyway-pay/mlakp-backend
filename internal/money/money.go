package money

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const MinorUnitsPerMajor int64 = 100

var (
	ErrInvalidAmount     = errors.New("amount is invalid")
	ErrAmountNotPositive = errors.New("amount must be greater than zero")
	ErrInvalidSplit      = errors.New("split is invalid")
	ErrSplitMismatch     = errors.New("split total must equal amount")
)

func ParseMinor(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, ErrInvalidAmount
	}
	if strings.HasPrefix(value, "-") || strings.HasPrefix(value, "+") {
		return 0, ErrInvalidAmount
	}

	majorText, minorText, ok := strings.Cut(value, ".")
	if !ok {
		minorText = "00"
	} else if minorText == "" || strings.Contains(minorText, ".") || len(minorText) > 2 {
		return 0, ErrInvalidAmount
	}

	if majorText == "" || !isDigits(majorText) {
		return 0, ErrInvalidAmount
	}
	if !isDigits(minorText) {
		return 0, ErrInvalidAmount
	}
	for len(minorText) < 2 {
		minorText += "0"
	}

	major, err := strconv.ParseInt(majorText, 10, 64)
	if err != nil {
		return 0, ErrInvalidAmount
	}
	minor, err := strconv.ParseInt(minorText, 10, 64)
	if err != nil {
		return 0, ErrInvalidAmount
	}
	if major > (math.MaxInt64-minor)/MinorUnitsPerMajor {
		return 0, ErrInvalidAmount
	}

	return major*MinorUnitsPerMajor + minor, nil
}

func FormatMinor(amountMinor int64) string {
	sign := ""
	if amountMinor < 0 {
		sign = "-"
		amountMinor = -amountMinor
	}

	return fmt.Sprintf("%s%d.%02d", sign, amountMinor/MinorUnitsPerMajor, amountMinor%MinorUnitsPerMajor)
}

func ValidatePositive(amountMinor int64) error {
	if amountMinor <= 0 {
		return ErrAmountNotPositive
	}

	return nil
}

func SplitEqual(totalMinor int64, participants int) ([]int64, error) {
	if totalMinor <= 0 || participants <= 0 {
		return nil, ErrInvalidSplit
	}

	base := totalMinor / int64(participants)
	remainder := totalMinor % int64(participants)
	if base == 0 {
		return nil, ErrInvalidSplit
	}

	shares := make([]int64, participants)
	for i := range shares {
		shares[i] = base
		if int64(i) >= int64(participants)-remainder {
			shares[i]++
		}
	}

	return shares, nil
}

func ValidateManualSplit(totalMinor int64, shares []int64) error {
	if totalMinor <= 0 || len(shares) == 0 {
		return ErrInvalidSplit
	}

	var sum int64
	for _, share := range shares {
		if share <= 0 {
			return ErrInvalidSplit
		}
		if sum > math.MaxInt64-share {
			return ErrInvalidSplit
		}
		sum += share
	}

	if sum != totalMinor {
		return ErrSplitMismatch
	}

	return nil
}

func isDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}
