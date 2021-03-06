package types

import (
	//"log"
	"os"

	"gopkg.in/yaml.v2"
)

type Match struct {
	Id             string
	Model          string
	Uid            string
	MinPosition    int
	MaxMatchEndPos int // For Not constraint, match should not go above this limit
	Spacer         bool
	Start          int
	End            int
	Sub            int
	Indel          int
	Info           string
	Children       []Match
	MinRepeat      int
	MaxRepeat      int
	Duration       int64 // Time.UnixNano
	From           []From
	YetToBeDefined []string
	//NeedCassie bool
	SavedAs   string
	IsModel   bool
	Overlap   bool
	SpacerVar bool
	Reverse   bool
}

func NewMatch() Match {
	match := Match{}
	match.Start = -1
	match.End = -1
	return match
}

func CheckMatches(matches []Match, start_pos int, prev_spacer bool) bool {
	// Checks that start/end of consecutive matches corresponds
	isFake := os.Getenv("LOGOL_FAKE")
	if isFake != "" {
		return true
	}

	end_pos := 0
	for index, m := range matches {
		if m.SpacerVar {
			prev_spacer = true
			continue
		}
		if m.Start == -1 || m.End == -1 {
			logger.Errorf("Humm, something wrong occured, a variable is still not defined %s.%s: %s", m.Model, m.Id, m.Uid)
			return false
		}
		if index == 0 && start_pos > 0 && m.Start != start_pos {
			logger.Errorf("position does not fit with starting position %d", start_pos)
			return false
		}
		if !prev_spacer {
			if m.Overlap {
				if m.End < end_pos {
					logger.Errorf("position does not fit with overlap")
					return false
				}
			} else if m.Start == end_pos || end_pos == 0 {
				end_pos = m.End
			} else {
				logger.Errorf("position does not fit with previous match end")
				return false
			}
		} else {
			if m.Start >= end_pos {
				end_pos = m.End
			} else {
				logger.Errorf("position does not fit with previous match end")
				return false
			}
		}
		if len(m.Children) > 0 {
			CheckMatches(m.Children, m.Start, prev_spacer)
		}
		prev_spacer = false

	}
	return true
}

func (m Match) IsMaxPosExceeded() bool {
	if m.MaxMatchEndPos > 0 && m.End > m.MaxMatchEndPos {
		return true
	}
	return false
}

func (m Match) GetById(model string, id string) ([]Match, bool) {
	// Parse match and match Children to find elements matching model and var name
	result := make([]Match, 0)
	found := false
	if m.Model == model && m.Id == id {
		result = append(result, m)
		found = true
	} else {
		if len(m.Children) > 0 {
			for _, subm := range m.Children {
				childMatches, found := subm.GetById(model, id)
				if found {
					found = true
					result = append(result, childMatches...)
				}
			}
		}
	}
	return result, found
}

func (m Match) GetByUid(uid string) (Match, bool) {
	// Parse match and match Children to find elements matching uid

	if m.Uid == uid {
		return m, true
	} else {
		if len(m.Children) > 0 {
			for _, subm := range m.Children {
				childMatch, found := subm.GetByUid(uid)
				if found {
					return childMatch, true
				}
			}
			return Match{}, false
		} else {
			return Match{}, false
		}
	}
}

func (m Match) Dumps() (out []byte, err error) {
	return yaml.Marshal(&m)
}

func (m Match) Clone() (clone Match) {
	clone = NewMatch()
	clone.Id = m.Id
	clone.Model = m.Model
	clone.Uid = m.Uid
	clone.MinPosition = m.MinPosition
	clone.Start = m.Start
	clone.End = m.End
	clone.Sub = m.Sub
	clone.Indel = m.Indel
	clone.Info = m.Info
	for _, child := range m.Children {
		clone.Children = append(clone.Children, child.Clone())
	}
	clone.MinRepeat = m.MinRepeat
	clone.MaxRepeat = m.MaxRepeat
	clone.Duration = m.Duration
	clone.YetToBeDefined = m.YetToBeDefined
	clone.Spacer = m.Spacer
	//clone.NeedCassie = m.NeedCassie
	return clone
}
