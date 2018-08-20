package logol

import (
    "fmt"
    "log"
    //"io/ioutil"
    "os"
    "strconv"
    "strings"
    lru "github.com/hashicorp/golang-lru"
)


type Sequence struct {
    Path string
    Size int
    Content string
}

type SequenceLru struct {
    Lru *lru.Cache
    Sequence Sequence
}

// Initialize a Sequence
func NewSequence(path string) (seq Sequence){
    seq = Sequence{}
    seq.Path = path
    file, err := os.Open(path)
    if err != nil {
        log.Fatalf("%s: %s", "failed to open sequence", err)
        panic(fmt.Sprintf("%s: %s", "failed to open sequence", err))
    }
    defer file.Close()
    stat, _ := file.Stat()
    seq.Size = int(stat.Size())
    return seq
}

// Initialize a LRU cache for sequence
func NewSequenceLru(sequence Sequence) (seq SequenceLru){
    log.Printf("Initialize sequence LRU")
    seq = SequenceLru{}
    seq.Sequence = sequence
    seq.Lru, _ = lru.New(10)
    return seq
}

// Get content from sequence using LRU cache
func (s SequenceLru) GetContent(start int, end int) (content string) {
    //log.Printf("Search in sequence %d:%d", start, end)
    keys := s.Lru.Keys()
    sRange := ""
    sStart := 0
    sEnd := 0
    for _, key := range keys {
        //log.Printf("Cache content: %s", key.(string))
        r := strings.Split(key.(string), ".")
        sStart, _ = strconv.Atoi(r[0])
        sEnd, _ = strconv.Atoi(r[1])
        if start >= sStart && end <= sEnd {
            sRange = key.(string)
            break
        }
    }
    if sRange != "" {
        log.Printf("Load sequence from cache")
        cache, _ := s.Lru.Get(sRange)
        seqPart := cache.(string)
        log.Printf("cache %s, %d, %d", seqPart, start, end)
        content = seqPart[start - sStart:end - start]
        // content := subpart of cache
        return content
    } else {
        //log.Printf("Read sequence file to extract content")
        file, _ := os.Open(s.Sequence.Path)
        defer file.Close()
        if end > s.Sequence.Size {
            end = s.Sequence.Size - 1
        }
        bufferSize := 10000
        if start + bufferSize > s.Sequence.Size {
            bufferSize = end - start
        }
        if end - start > bufferSize {
            bufferSize = end - start
        }
        buffer := make([]byte, bufferSize)
        log.Printf("Load from sequence %d, %d", start, end)
        file.ReadAt(buffer, int64(start))
        // get content
        content := string(buffer)
        key := fmt.Sprintf("%d.%d", start, end)
        //log.Printf("Save in LRU %s", key)
        s.Lru.Add(key, content)
        return content
    }
}

/*
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
}*/
