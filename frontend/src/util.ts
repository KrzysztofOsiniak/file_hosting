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