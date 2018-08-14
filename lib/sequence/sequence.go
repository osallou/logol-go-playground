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
    content = sequence[start:end]
    return content
}
