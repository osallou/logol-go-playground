package logol


import (
        "path/filepath"
        "testing"
        // "log"
)


func TestSequenceAccess(t *testing.T) {
    path := filepath.Join("testdata", "sequence.txt")
    seq := NewSequence(path)
    seqLru := NewSequenceLru(seq)
    content := seqLru.GetContent(1,3)
    if content != "cc" {
        t.Errorf("Invalid result")
    }
    content = seqLru.GetContent(0,3)
    if content != "ccc" {
        t.Errorf("Invalid result")
    }
    content = seqLru.GetContent(10,30)
    if content != "cgtttttt" {
        t.Errorf("Invalid result: cgtttttt vs %s", content)
    }

}
