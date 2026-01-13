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
    return Math.floor(size*100)/100
}

// If not zero, leftover is the size of the last part in partCount.
// Minimum part size (in aws s3) is 5MiB (not counting the last part).
export function splitFile(bytes: number): {partCount: number, partSize: number, leftover: number} {
	if (bytes <= Math.pow(10, 7)) {
		return {partCount: 1, partSize: 0, leftover: bytes}
	}
	if (bytes <= Math.pow(10, 8)) {
		let partSize = 6 * Math.pow(10, 6)
		let partCount = (bytes - bytes%partSize) / partSize
		if (bytes%partSize != 0) {
			partCount++
		}
		return {partCount: partCount, partSize: partSize, leftover: bytes % partSize}
	}
	if (bytes <= Math.pow(10, 9)) {
		let partSize = Math.pow(10, 7)
		let partCount = (bytes - bytes%partSize) / partSize
		if (bytes%partSize != 0) {
			partCount++
		}
		return {partCount: partCount, partSize: partSize, leftover: bytes % partSize}
	}
	if (bytes <= Math.pow(10, 11)){
		let partSize = Math.pow(10, 8)
		let partCount = (bytes - bytes%partSize) / partSize
		if (bytes%partSize != 0) {
			partCount++
		}
		return {partCount: partCount, partSize: partSize, leftover: bytes % partSize}
	}
	if (bytes <= 5*Math.pow(10, 12)) {
		let partSize = 5 * Math.pow(10, 8)
		let partCount = (bytes - bytes%partSize) / partSize
		if (bytes%partSize != 0) {
			partCount++
		}
		return {partCount: partCount, partSize: partSize, leftover: bytes % partSize}
	}
	return {partCount: 0, partSize: 0, leftover: 0}
}