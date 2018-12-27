package logol

import (
	"gopkg.in/yaml.v2"
	//"encoding/json"
)

type Event struct {
	Step int
}

type From struct {
	Model    string
	Variable string
	Uid      string
}

type Result struct {
	Outfile          string
	SequenceFile     string
	RunIndex         int
	Uid              string
	MsgTo            string
	Model            string
	ModelVariable    string
	From             []From
	CallbackUid      string
	Previous         string
	Matches          []Match
	PrevMatches      [][]Match // store previous main model matches when using serial models
	ContextVars      []map[string]Match
	Spacer           bool
	Overlap          bool
	Context          [][]Match
	Step             int
	Position         int
	Iteration        int
	Param            []Match
	YetToBeDefined   []Match // temporary store matches depending on variables not yet defined
	ExpectNoMatch    bool
	ExpectNoMatchVar Match
	ExpectNoMatchUID string
}

func NewResult() Result {
	result := Result{}
	result.From = make([]From, 0)
	result.Matches = make([]Match, 0)
	result.PrevMatches = make([][]Match, 0)
	result.ContextVars = make([]map[string]Match, 0)
	result.Context = make([][]Match, 0)
	result.Param = make([]Match, 0)
	return result
}

func (m Result) Dumps() (out []byte, err error) {
	return yaml.Marshal(&m)
}

type matchPosition struct {
	MinPos     int
	PreSpacer  bool
	MaxPos     int
	PostSpacer bool
	Gotcha     bool
}

func findMatchSurroundingPositions(uid string, matches []Match, position matchPosition) (positions matchPosition) {
	nbmatches := len(matches)
	if nbmatches == 0 {
		return position
	}
	for _, match := range matches {
		//pos_json, _ := json.Marshal(position)
		if match.Uid == uid {
			// found match
			position.Gotcha = true
			continue
		} else {
			if !position.Gotcha {
				// check in children
				position = findMatchSurroundingPositions(uid, match.Children, position)
				if position.Gotcha {
					if position.MaxPos == -1 {
						// Found match but not its next position
						continue
					} else {
						// Found everything, exiting
						return position
					}
				}
			}
		}

		if !position.Gotcha {
			// Looking for prev position
			if match.SpacerVar {
				position.PreSpacer = true
				continue
			}
			if match.End > -1 {
				position.MinPos = match.End
				position.PreSpacer = false
				continue
			} else {
				if len(match.Children) == 0 {
					position.PreSpacer = true
					continue
				}
				position = findMatchSurroundingPositions(uid, match.Children, position)
				if position.MinPos == -1 {
					position.PreSpacer = true
				}
			}
		}

		if position.Gotcha {
			// Look for next position
			if match.SpacerVar {
				position.PostSpacer = true
				continue
			}
			if match.Start > -1 {
				position.MaxPos = match.Start
				break
			} else {
				if len(match.Children) == 0 {
					position.PostSpacer = true
					continue
				}
				position = findMatchSurroundingPositions(uid, match.Children, position)
				if position.MaxPos == -1 {
					position.PostSpacer = true
				}
				//position.PostSpacer = true
				continue
			}
		}
	}
	return position
}
func (r Result) FindSurroundingPositions(uid string) (min_pos int, pre_spacer bool, max_pos int, post_spacer bool) {
	logger.Debugf("FindSurroundingPositions:%s", uid)
	mpos := matchPosition{}
	mpos.MinPos = -1
	mpos.MaxPos = -1
	mpos.PreSpacer = true
	position := findMatchSurroundingPositions(uid, r.Matches, mpos)
	if !position.Gotcha {
		for _, matches := range r.PrevMatches {
			position = findMatchSurroundingPositions(uid, matches, mpos)
			if position.Gotcha {
				break
			}
		}
	}
	if position.Gotcha {
		return position.MinPos, position.PreSpacer, position.MaxPos, position.PostSpacer
	}
	return 0, true, 0, true
}

func (m Result) GetFirstMatchAnalysable() int {
	// Get first match in result that can be resolved (do not depend itself on other unknown matches)
	if len(m.YetToBeDefined) == 0 {
		return -2
	}
	uids := make(map[string]bool)
	for _, match := range m.YetToBeDefined {
		uids[match.Uid] = true
	}
	for i, match := range m.YetToBeDefined {
		canBeAnalysed := true
		for _, uid := range match.YetToBeDefined {
			_, present := uids[uid]
			if present {
				canBeAnalysed = false
				break
			}
		}
		if canBeAnalysed {
			return i
		}
	}
	return -1
}
