package logol

import (
    "strings"
    // "log"
)


type BioString interface {
    IsExact(s2 string) bool
    IsApproximate(s2 string, subst int, indel int) (bool, int, int)
    Reverse() string
    Complement() string
}


type DnaString struct {
    Value string
    // List of morphisms ("a" -> "cg", "g" -> "t",...)
    AllowedMorphisms map[string][]string
}

func (s DnaString) Complement() (string){
    complement := ""
    sLen := len(s.Value)
    for i:=0;i<sLen;i++ {
        switch s.Value[i:i+1] {
        case "a":
            complement += "t"
        case "c":
            complement += "g"
        case "g":
            complement += "c"
        case "t":
            complement += "a"
        }
    }
    s.Value = complement
    return s.Value
}

func (s DnaString) Reverse() (string){
    reverse := ""
    sLen := len(s.Value)
    for i:=0;i<sLen;i++ {
        reverse += s.Value[sLen - i - 1:sLen - i]
    }
    s.Value = reverse
    return s.Value
}

func (s DnaString) IsExact(s2 string) bool {
    chain1 := strings.ToLower(s.Value)
    chain2 := strings.ToLower(s2)

    s1Len := len(chain1)
    s2Len := len(chain2)
    if s1Len != s2Len {
        return false
    }
    for i:=0; i < s1Len; i++ {
        chain2Char := chain2[i:i+1]
        morph, ok := s.AllowedMorphisms[chain2Char]
        if ok {
            gotcha := false
            for _, charMap := range morph {
                    if chain1[i:i+1] == charMap {
                        gotcha = true
                        break
                    }
            }
            if ! gotcha {
                return false
            }
        } else {
            if chain1[i:i+1] == "n" || chain2Char == "n" {
                continue
            } else if chain1[i] != chain2[i] {
                return false
            }
        }
    }
    return true
}

func (s DnaString) IsApproximate(s2 string, subst int, indel int) (bool, int, int) {
    return true, 0, 0
}

func IsBioExact(b1 BioString, b2 string) bool {
    return b1.IsExact(b2)
}
