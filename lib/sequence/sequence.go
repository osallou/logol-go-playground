package logol

import (
    "log"
    "io/ioutil"
)


type Sequence struct {
    Path string
    Size int
    Content string
}

func (s Sequence) GetSequence() (string) {
    if s.Content == "" {
        content, err := ioutil.ReadFile(s.Path)
        if err != nil {
            log.Printf("Could not read sequence %s", s.Path)
            return ""
        }
        return string(content)
    }else {
        return s.Content
    }
}

func (s Sequence) GetContent(start int, end int) (content string) {
    sequence := s.GetSequence()
    if sequence == ""{
        return ""
    }
    log.Printf("Extract sequence value from %d:%d", start, end)
    seqLen := len(sequence)
    if (start < 0 || end < 0 || start >= seqLen || end >= seqLen) {
        log.Printf("Cannot read sequence value %d, %d", start, end)
        return ""
    }
    content = sequence[start:end]
    return content
}

/*
func GetSequence(filePath string) (string) {
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
}*/
