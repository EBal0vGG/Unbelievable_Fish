package identity

import (
	"regexp"
	"strconv"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}$`)

func isBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}

func normalizeLogin(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isValidEmail(email string) bool {
	email = normalizeLogin(email)
	if isBlank(email) {
		return false
	}

	return emailRegex.MatchString(email)
}

func isValidINN(inn string) bool {
	inn = strings.TrimSpace(inn)
	if !isDigits(inn) {
		return false
	}

	switch len(inn) {
	case 10:
		checksum := []int{2, 4, 10, 3, 5, 9, 4, 6, 8}
		return digitChecksum(inn[:9], checksum, 11, 10) == int(inn[9]-'0')
	case 12:
		firstChecksum := []int{7, 2, 4, 10, 3, 5, 9, 4, 6, 8}
		secondChecksum := []int{3, 7, 2, 4, 10, 3, 5, 9, 4, 6, 8}

		return digitChecksum(inn[:10], firstChecksum, 11, 10) == int(inn[10]-'0') &&
			digitChecksum(inn[:11], secondChecksum, 11, 10) == int(inn[11]-'0')
	default:
		return false
	}
}

func isValidOGRN(ogrn string) bool {
	ogrn = strings.TrimSpace(ogrn)
	if !isDigits(ogrn) {
		return false
	}

	switch len(ogrn) {
	case 13:
		base, err := strconv.ParseInt(ogrn[:12], 10, 64)
		if err != nil {
			return false
		}

		return int(base%11%10) == int(ogrn[12]-'0')
	case 15:
		base, err := strconv.ParseInt(ogrn[:14], 10, 64)
		if err != nil {
			return false
		}

		return int(base%13%10) == int(ogrn[14]-'0')
	default:
		return false
	}
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

func digitChecksum(value string, weights []int, mod int, remainderMod int) int {
	sum := 0
	for i := range value {
		sum += int(value[i]-'0') * weights[i]
	}

	return (sum % mod) % remainderMod
}
