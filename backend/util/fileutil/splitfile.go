package fileutil

// If not zero, leftover is the size of the last part in partCount.
// Minimum part size (in aws s3) is 5MiB (not counting the last part).
func SplitFile(bytes int) (partCount int, partSize int, leftover int) {
	if bytes <= intPow(10, 7) {
		return 1, 0, bytes
	}
	if bytes <= intPow(10, 8) {
		partSize := 6 * intPow(10, 6)
		partCount := (bytes - bytes%partSize) / partSize
		if bytes%partSize != 0 {
			partCount++
		}
		return partCount, partSize, bytes % partSize
	}
	if bytes <= intPow(10, 9) {
		partSize := intPow(10, 7)
		partCount := (bytes - bytes%partSize) / partSize
		if bytes%partSize != 0 {
			partCount++
		}
		return partCount, partSize, bytes % partSize
	}
	if bytes <= intPow(10, 11) {
		partSize := intPow(10, 8)
		partCount := (bytes - bytes%partSize) / partSize
		if bytes%partSize != 0 {
			partCount++
		}
		return partCount, partSize, bytes % partSize
	}
	if bytes <= 5*intPow(10, 12) {
		partSize := 5 * intPow(10, 8)
		partCount := (bytes - bytes%partSize) / partSize
		if bytes%partSize != 0 {
			partCount++
		}
		return partCount, partSize, bytes % partSize
	}
	return 0, 0, 0
}

func intPow(n, m int) int {
	if m == 0 {
		return 1
	}
	if m == 1 {
		return n
	}

	result := n
	for i := 2; i <= m; i++ {
		result *= n
	}
	return result
}
