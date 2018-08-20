package logol


import (
        "path/filepath"
        "testing"
        logol "org.irisa.genouest/logol/lib/types"
        "log"
)


func TestFindAny(t *testing.T) {
    path := filepath.Join("testdata", "sequence.txt")
    seqUtils := NewSearchUtils(path)
    contextVars := make(map[string]logol.Match)
    grammar := logol.Grammar{}
    match := logol.Match{}
    match.MinPosition = 2
    mch := make(chan logol.Match)
    nbMatches := 0
    log.Printf("Search any match")
    go seqUtils.FindAny(mch, grammar, match, "mod1", "var1", 4, 4, contextVars,false)
    for m := range mch {
        log.Printf("Got res %d:%d", m.Start, m.End)
        log.Printf("Found %s", seqUtils.SequenceHandler.GetContent(m.Start, m.End))
        nbMatches += 1
        if m.Start != 2 && m.End != 6 {
            t.Errorf("Invalid result")
            break
        }
    }

    if nbMatches != 1 {
        t.Errorf("Invalid number of result, %d", nbMatches)
    }
}

func TestFindAnyMultipleSize(t *testing.T) {
    path := filepath.Join("testdata", "sequence.txt")
    seqUtils := NewSearchUtils(path)
    contextVars := make(map[string]logol.Match)
    grammar := logol.Grammar{}
    match := logol.Match{}
    match.MinPosition = 2
    mch := make(chan logol.Match)
    nbMatches := 0
    log.Printf("Search any match")
    go seqUtils.FindAny(mch, grammar, match, "mod1", "var1", 4, 6, contextVars, false)
    for m := range mch {
        log.Printf("Got res %d:%d", m.Start, m.End)
        log.Printf("Found %s", seqUtils.SequenceHandler.GetContent(m.Start, m.End))
        nbMatches += 1
    }

    if nbMatches != 3 {
        t.Errorf("Invalid number of result %d", nbMatches)
    }
}


func TestFindAnyWithSpacer(t *testing.T) {
    path := filepath.Join("testdata", "sequence.txt")
    seqUtils := NewSearchUtils(path)
    contextVars := make(map[string]logol.Match)
    grammar := logol.Grammar{}
    match := logol.Match{}
    match.MinPosition = 2
    mch := make(chan logol.Match)
    nbMatches := 0
    log.Printf("Search any match")
    go seqUtils.FindAny(mch, grammar, match, "mod1", "var1", 4, 4, contextVars, true)
    for m := range mch {
        log.Printf("Got res %d:%d", m.Start, m.End)
        log.Printf("Found %s", seqUtils.SequenceHandler.GetContent(m.Start, m.End))
        nbMatches += 1
        if m.Start < 2 && m.End != m.Start + 4 {
            t.Errorf("Invalid result %d:%d", m.Start, m.End)
            break
        }
    }

    if nbMatches != 13 {
        t.Errorf("Invalid number of result %d", nbMatches)
    }
}
