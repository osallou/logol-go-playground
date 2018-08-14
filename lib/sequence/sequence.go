package logol

import (
    "log"
)

func GetSequence() (string) {
    return "ccccaaaaaacgtttttt";
}


func GetContent(start int, end int) (content string){
    sequence := GetSequence()
    log.Printf("Extract sequence value from %d:%d", start, end)
    seqLen := len(sequence)
    if (start < 0 || end < 0 || start >= seqLen || end >= seqLen) {
        log.Printf("Cannot read sequence value %d, %d", start, end)
        return ""
    }
    content = sequence[start:end]
    return content
}
