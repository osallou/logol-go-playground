package logol


import (
        "testing"
        //"log"
)

func TestBioIsExact(t *testing.T) {
    b1 := DnaString{}
    b1.Value = "acgt"
    b2 := "acgt"
    if ! IsBioExact(b1, b2) {
        t.Errorf("Invalid match")
    }

    b1 = DnaString{}
    b1.Value = "acgt"
    b2 = "cgtt"
    if IsBioExact(b1, b2) {
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
    if IsBioExact(b1, b2) {
        t.Errorf("Invalid match")
    }
    b1.Value = "aagt"
    b2 = "ccgt"
    if ! IsBioExact(b1, b2) {
        t.Errorf("Invalid match")
    }
}
