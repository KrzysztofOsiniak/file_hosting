export function getUnit(size: number | null) {
    if(size === null) {
        return "B"
    }
    let i = 0
    while(size >= 1000) {
        i++
        size = size/1000
    }
    switch(i) {
        case 0:
            return "B"
        case 1:
            return "KB"
        case 2:
            return "MB"
        case 3:
            return "GB"
        case 4:
            return "TB"
    }
}

export function getUnitSize(size: number | null) {
    if(size === null) {
        return 0
    }
    while(size >= 1000) {
        size = size/1000
    }
    return size
}

// If not zero, leftover is the size of the last part in partCount.
// Minimum part size (in aws s3) is 5MiB (not counting the last part).
// export function splitFile(bytes: number) (partCount int, partSize int, leftover int) {
// 	if bytes <= intPow(10, 7) {
// 		return 1, 0, bytes
// 	}
// 	if bytes <= intPow(10, 8) {
// 		partSize := 6 * intPow(10, 6)
// 		partCount := (bytes - bytes%partSize) / partSize
// 		if bytes%partSize != 0 {
// 			partCount++
// 		}
// 		return partCount, partSize, bytes % partSize
// 	}
// 	if bytes <= intPow(10, 9) {
// 		partSize := intPow(10, 7)
// 		partCount := (bytes - bytes%partSize) / partSize
// 		if bytes%partSize != 0 {
// 			partCount++
// 		}
// 		return partCount, partSize, bytes % partSize
// 	}
// 	if bytes <= intPow(10, 11) {
// 		partSize := intPow(10, 8)
// 		partCount := (bytes - bytes%partSize) / partSize
// 		if bytes%partSize != 0 {
// 			partCount++
// 		}
// 		return partCount, partSize, bytes % partSize
// 	}
// 	if bytes <= 5*intPow(10, 12) {
// 		partSize := 5 * intPow(10, 8)
// 		partCount := (bytes - bytes%partSize) / partSize
// 		if bytes%partSize != 0 {
// 			partCount++
// 		}
// 		return partCount, partSize, bytes % partSize
// 	}
// 	return 0, 0, 0
// }