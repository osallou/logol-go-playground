package logol


import (
        "path/filepath"
        "testing"
        logol "org.irisa.genouest/logol/lib/types"
        "log"
        //"encoding/json"
)

func TestFindApproximate(t *testing.T){

    res:= IsApproximate("acgt", "acgt", 0, 0, 0, 0)
    resLen := len(res)
    if resLen == 0 {
        t.Errorf("Invalid result")
    }
    res= IsApproximate("acgt", "aggt", 0, 1, 0, 0)
    resLen = len(res)
    if resLen == 0 {
        t.Errorf("Invalid result")
    }

    res= IsApproximate("acgt", "accgt", 0, 0, 0, 1)
    resLen = len(res)
    if resLen == 0 {
        t.Errorf("Invalid result")
    }
    res= IsApproximate("acgt", "acgtttt", 0, 0, 0, 1)
    resLen = len(res)
    if resLen == 0 {
        t.Errorf("Invalid result")
    }
    res= IsApproximate("acgt", "atttt", 0, 0, 0, 1)
    resLen = len(res)
    if resLen != 0 {
        t.Errorf("Invalid result")
    }
    res= IsApproximate("acgt", "atctttt", 0, 1, 0, 1)
    resLen = len(res)
    if resLen == 0 {
        t.Errorf("Invalid result")
    }
    //json_res, _ := json.Marshal(res)
    //log.Printf("Result approximate: %s", json_res)

}

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
