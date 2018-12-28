// Package sequence manage the different sequence search methods to find a match according to grammar
package sequence

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	cassie "github.com/osallou/cassiopee-go"
	logol "github.com/osallou/logol-go-playground/lib/types"
)

var once sync.Once

// CassieIndexerInstance is an instance of CassieIndexer
var CassieIndexerInstance *cassie.CassieIndexer

// GetCassieIndexer returns a new CassieIndexerInstance
func GetCassieIndexer(grammar logol.Grammar) *cassie.CassieIndexer {
	once.Do(func() {
		instance := cassie.NewCassieIndexer(grammar.Sequence)
		if grammar.Options != nil {
			instance.SetMax_index_depth(grammar.Options["MAX_PATTERN_LENGTH"])
		} else {
			instance.SetMax_index_depth(1000)
		}
		//instance.SetMax_depth(0)
		instance.SetDo_reduction(true)
		logger.Infof("Indexing sequence %s", grammar.Sequence)
		instance.Index()
		CassieIndexerInstance = &instance
	})
	return CassieIndexerInstance
}

// Sequence describes a sequence to analyze
type Sequence struct {
	Path    string
	Size    int
	Content string
}

// SequenceLru is a LRU cache for sequence parts
type SequenceLru struct {
	Lru      *lru.Cache
	Sequence Sequence
}

// NewSequence initialize a Sequence
func NewSequence(path string) (seq Sequence) {
	seq = Sequence{}
	seq.Path = path
	file, err := os.Open(path)
	if err != nil {
		logger.Errorf("%s: %s", "failed to open sequence", err)
		panic(fmt.Sprintf("%s: %s", "failed to open sequence", err))
	}
	defer file.Close()
	stat, _ := file.Stat()
	seq.Size = int(stat.Size())
	return seq
}

// NewSequenceLru initialize a LRU cache for sequence
func NewSequenceLru(sequence Sequence) (seq SequenceLru) {
	logger.Debugf("Initialize sequence LRU")
	seq = SequenceLru{}
	seq.Sequence = sequence
	seq.Lru, _ = lru.New(10)
	return seq
}

// GetContent get content from sequence using LRU cache
func (s SequenceLru) GetContent(start int, end int) (content string) {
	logger.Debugf("Search in sequence %d:%d", start, end)
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
	logger.Debugf("seq start: %d, end: %d", sStart, sEnd)
	if sRange != "" {
		//log.Printf("Load sequence from cache")
		cache, _ := s.Lru.Get(sRange)
		seqPart := cache.(string)
		logger.Debugf("cache, %d, %d", start, end)
		content = seqPart[start-sStart : end-sStart]
		return content
	}
	file, _ := os.Open(s.Sequence.Path)
	defer file.Close()
	if end > s.Sequence.Size {
		end = s.Sequence.Size - 1
	}
	bufferSize := 10000
	if start+bufferSize > s.Sequence.Size {
		bufferSize = end - start
	}
	if end-start > bufferSize {
		bufferSize = end - start
	}
	buffer := make([]byte, bufferSize)
	logger.Debugf("Load from sequence %d, %d", start, end)
	file.ReadAt(buffer, int64(start))
	// get content
	content = string(buffer)
	key := fmt.Sprintf("%d.%d", start, end)
	//logger.Debugf("Save in LRU %s", key)
	s.Lru.Add(key, content)
	return content

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
