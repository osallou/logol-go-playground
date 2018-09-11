package logol


import (
        "path/filepath"
        "io/ioutil"
        "testing"
        logol "github.com/osallou/logol-go-playground/lib/types"
        "log"
        "encoding/json"
)

func TestGetMorphisms(t *testing.T){
    grammarpath := filepath.Join("testdata", "grammar.txt")
    g, _ := ioutil.ReadFile(grammarpath)
    err, grammar := logol.LoadGrammar([]byte(g))
    if err != nil {
            log.Fatalf("error: %v", err)
    }
    variable := logol.Variable{}
    variable.String_constraints.Morphism = "wc"
    m := variable.GetMorphism(grammar.Morphisms)
    json_msg, _ := json.Marshal(m)
    //log.Printf("## %s", json_msg)
    res, ok := m.Morph["a"]
    if !ok {
        t.Errorf("Invalid result %s", json_msg)
    }
    if res[0] != "t" {
        t.Errorf("Invalid result %s", json_msg)
    }
}

func TestIsApproximate(t *testing.T){
    ds := NewDnaString("acgt")
    res:= IsApproximate(&ds, "acgt", 0, 0, 0, 0, 0)
    resLen := len(res)
    if resLen == 0 {
        t.Errorf("Invalid result")
    }
    ds = NewDnaString("acgt")
    res= IsApproximate(&ds, "aggt", 0, 1, 0, 0, 0)
    resLen = len(res)
    if resLen == 0 {
        t.Errorf("Invalid result")
    }

    ds = NewDnaString("acgt")
    res= IsApproximate(&ds, "accgt", 0, 0, 0, 0, 1)
    resLen = len(res)
    if resLen == 0 {
        t.Errorf("Invalid result")
    }

    ds = NewDnaString("acgt")
    res= IsApproximate(&ds, "acgtttt", 0, 0, 0, 0, 1)
    resLen = len(res)
    if resLen == 0 {
        t.Errorf("Invalid result")
    }

    ds = NewDnaString("acgt")
    res= IsApproximate(&ds, "atttt", 0, 0, 0, 0, 1)
    resLen = len(res)
    if resLen != 0 {
        t.Errorf("Invalid result")
    }

    ds = NewDnaString("acgt")
    res= IsApproximate(&ds, "atctttt", 0, 1, 0, 0, 1)
    resLen = len(res)
    if resLen == 0 {
        t.Errorf("Invalid result")
    }
    //json_res, _ := json.Marshal(res)
    //log.Printf("Result approximate: %s", json_res)

}

func TestFindApproximate(t *testing.T){
    path := filepath.Join("testdata", "sequence.txt")
    grammarpath := filepath.Join("testdata", "grammar.txt")
    seqUtils := NewSearchUtils(path)
    contextVars := make(map[string]logol.Match)
    g, _ := ioutil.ReadFile(grammarpath)
    err, grammar := logol.LoadGrammar([]byte(g))
    if err != nil {
            log.Fatalf("error: %v", err)
    }
    match := logol.Match{}
    match.MinPosition = 2
    mch := make(chan logol.Match)
    nbMatches := 0
    log.Printf("Search approximate match")
    go seqUtils.FindApproximate(mch, grammar, match, "mod1", "var2", contextVars, false, 0, 0)
    for m := range mch {
        log.Printf("Got res %d:%d", m.Start, m.End)
        log.Printf("Found %s", seqUtils.SequenceHandler.GetContent(m.Start, m.End))
        nbMatches += 1
        if m.Start != 2 && m.End != 6 {
            t.Errorf("Invalid result")
            break
        }
    }
    if nbMatches != 0 {
        t.Errorf("Invalid number of result, %d", nbMatches)
    }

    mch = make(chan logol.Match)
    nbMatches = 0
    log.Printf("Search approximate match with sub")
    go seqUtils.FindApproximate(mch, grammar, match, "mod1", "var2", contextVars, false, 2, 0)
    for m := range mch {
        // log.Printf("Got res %d:%d", m.Start, m.End)
        log.Printf("Found %s", seqUtils.SequenceHandler.GetContent(m.Start, m.End))
        json_msg, _ := json.Marshal(m)
        log.Printf("Got %s", json_msg)
        nbMatches += 1
        if m.Start != 2 && m.End != 6 {
            t.Errorf("Invalid result")
            break
        }
    }
    if nbMatches != 1 {
        t.Errorf("Invalid number of result, %d", nbMatches)
    }

    mch = make(chan logol.Match)
    nbMatches = 0
    log.Printf("Search approximate match with indel")
    go seqUtils.FindApproximate(mch, grammar, match, "mod1", "var2", contextVars, false, 0, 2)
    for m := range mch {
        // log.Printf("Got res %d:%d", m.Start, m.End)
        log.Printf("Found %s", seqUtils.SequenceHandler.GetContent(m.Start, m.End))
        json_msg, _ := json.Marshal(m)
        log.Printf("Got %s", json_msg)
        nbMatches += 1
        if m.Start != 2 && m.End != 6 {
            t.Errorf("Invalid result")
            break
        }
    }
    if nbMatches != 2 {
        t.Errorf("Invalid number of result, %d", nbMatches)
    }

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
