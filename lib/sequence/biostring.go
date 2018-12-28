package sequence

import (
	"strings"
)

// BioString is an interface to manage biological sequences (dna, protein, ..)
type BioString interface {
	GetValue() string
	SetValue(string)
	IsExact(s2 string) bool
	IsApproximate(s2 string, subst int, indel int) (bool, int, int)
	Reverse() string
	Complement() string
	GetMorphisms(chan string)
	SetMorphisms(map[string][]string)
}

// DnaString defined a DNA sequence with optional allowed morphisms
type DnaString struct {
	Value string
	// List of morphisms ("a" -> "cg", "g" -> "t",...)
	AllowedMorphisms map[string][]string
}

// NewDnaString returns an instance of DnaString for input string
func NewDnaString(value string) (ds DnaString) {
	ds = DnaString{}
	ds.Value = value
	return ds
}

// GetValue returns DNA string
func (s DnaString) GetValue() string {
	return s.Value
}

// SetValue sets DNA string
func (s *DnaString) SetValue(str string) {
	s.Value = str
}

// getMorhpism returns all the morphisms for the string
func (s DnaString) getMorphism(ch chan string, part string, index int) {
	sLen := len(part)
	if index >= sLen {
		ch <- part
		return
	}
	sChar := s.Value[index : index+1]
	morphs, ok := s.AllowedMorphisms[sChar]
	if !ok {
		// continue
		s.getMorphism(ch, part, index+1)
	} else {
		// replace char at index by all possibilities and continue
		nbMorph := len(morphs)
		for l := 0; l < nbMorph; l++ {
			part = part[:index] + morphs[l] + part[index+1:]
			s.getMorphism(ch, part, index+1)
		}
	}
}

// GetMorphisms get all possible conversions with defined morphism
func (s DnaString) GetMorphisms(ch chan string) {
	if s.AllowedMorphisms == nil {
		logger.Debugf("No morphisms defined")
		ch <- s.Value
		close(ch)
		return
	}
	s.getMorphism(ch, s.Value, 0)
	close(ch)
}

// SetMorphisms sets allowed morphisms
func (s *DnaString) SetMorphisms(m map[string][]string) {
	s.AllowedMorphisms = m
}

// Complement returns the dna complement of the string
func (s DnaString) Complement() string {
	complement := ""
	sLen := len(s.Value)
	for i := 0; i < sLen; i++ {
		switch s.Value[i : i+1] {
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

// Reverse reverse a string
func (s DnaString) Reverse() string {
	reverse := ""
	sLen := len(s.Value)
	for i := 0; i < sLen; i++ {
		reverse += s.Value[sLen-i-1 : sLen-i]
	}
	s.Value = reverse
	return s.Value
}

// IsExact checks if both strings are equal with allowed morphisms
func (s DnaString) IsExact(s2 string) bool {
	chain1 := strings.ToLower(s.Value)
	chain2 := strings.ToLower(s2)
	logger.Debugf("Bio isexact %s  %s", chain1, chain2)

	s1Len := len(chain1)
	s2Len := len(chain2)
	if s1Len != s2Len {
		return false
	}
	for i := 0; i < s1Len; i++ {
		chain2Char := chain2[i : i+1]
		morph, ok := s.AllowedMorphisms[chain2Char]
		if ok {
			gotcha := false
			for _, charMap := range morph {
				if chain1[i:i+1] == charMap {
					gotcha = true
					break
				}
			}
			if !gotcha {
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

// IsApproximate checks if string are equal with possible errors
func (s DnaString) IsApproximate(s2 string, subst int, indel int) (bool, int, int) {
	return true, 0, 0
}

// IsBioExact checks if strings are equals (with allowed mutations)
func IsBioExact(b1 BioString, b2 string) bool {
	return b1.IsExact(b2)
}
