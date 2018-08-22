package logol


import (
        "testing"
        "log"
)

func TestMorphisms(t *testing.T) {
    b1 := DnaString{}
    b1.Value = "acgt"
    ch := make(chan string)
    go b1.GetMorphisms(ch)
    nbMatch := 0
    for s := range ch {
        nbMatch += 1
        log.Printf("Match: %s", s)
    }
    if nbMatch != 1 {
        t.Errorf("Invalid matches: %d", nbMatch)
    }
    ch = make(chan string)
    b1.AllowedMorphisms = make(map[string][]string)
    maps := make([]string, 2)
    maps[0] = "a"
    maps[1] = "g"
    b1.AllowedMorphisms["c"] = maps

    go b1.GetMorphisms(ch)
    nbMatch = 0
    for s := range ch {
        nbMatch += 1
        log.Printf("Match: %s", s)
    }
    if nbMatch != 2 {
        t.Errorf("Invalid matches: %d", nbMatch)
    }
}

func TestBioIsExact(t *testing.T) {
    b1 := DnaString{}
    b1.Value = "acgt"
    b2 := "acgt"
    if ! IsBioExact(&b1, b2) {
        t.Errorf("Invalid match")
    }

    b1 = DnaString{}
    b1.Value = "a"
    b2 = "a"
    if ! IsBioExact(&b1, b2) {
        t.Errorf("Invalid match")
    }

    b1 = DnaString{}
    b1.Value = "acgt"
    b2 = "cgtt"
    if IsBioExact(&b1, b2) {
        t.Errorf("Invalid match")
    }
}

func TestBioIsExactWithMorphism(t *testing.T) {
    b1 := DnaString{}
    b1.Value = "acgt"
    morphs := make([]string, 2)
    morphs[0] = "a"
    morphs[1] = "g"
    b1.AllowedMorphisms = make(map[string][]string)
    b1.AllowedMorphisms["c"] = morphs
    b2 := "acgt"
    if IsBioExact(&b1, b2) {
        t.Errorf("Invalid match")
    }
    b1.Value = "aagt"
    b2 = "ccgt"
    if ! IsBioExact(&b1, b2) {
        t.Errorf("Invalid match")
    }
}

func TestBioReverse(t *testing.T) {
    b1 := DnaString{}
    b1.Value = "aagg"
    rev := b1.Reverse()
    if rev != "ggaa" {
        t.Errorf("Invalid reverse")
    }
}

func TestBioComplement(t *testing.T) {
    b1 := DnaString{}
    b1.Value = "acgt"
    rev := b1.Complement()
    if rev != "tgca" {
        t.Errorf("Invalid complement")
    }
}
