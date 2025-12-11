package utils

// ValidateLuhn проверяет номер по алгоритму Луна.
func ValidateLuhn(number string) bool {
	var sum int
	parity := len(number) % 2
	for i, r := range number {
		if r < '0' || r > '9' {
			return false
		}
		digit := int(r - '0')
		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
	}
	return sum%10 == 0
}
