package logol


import (
        "testing"
        //logol "org.irisa.genouest/logol/lib/types"
        // "log"
)


func TestFindSurroundingPositions(t *testing.T) {
    result := NewResult()
    m := Match{}
    m.Start = 10
    m.End = 20
    m.Uid = "1"
    result.Matches = append(result.Matches, m)
    m = Match{}
    m.Start = 20
    m.End = 30
    m.Uid = "2"
    result.Matches = append(result.Matches, m)
    m = Match{}
    m.Start = -1
    m.End = -1
    m.Uid = "3"
    result.Matches = append(result.Matches, m)
    m = Match{}
    m.Start = 40
    m.End = 50
    m.Uid = "4"
    result.Matches = append(result.Matches, m)
    min_pos, pre_spacer, max_pos, post_spacer := result.FindSurroundingPositions("3")
    if min_pos != 30 || pre_spacer == true || max_pos != 40 || post_spacer == true {
        t.Errorf("Invalid result: %d %t %d %t", min_pos, pre_spacer, max_pos, post_spacer)
    }
}

func TestFindSurroundingPositionsWithPreUnknowns(t *testing.T) {
        result := NewResult()
        m := Match{}
        m.Start = 10
        m.End = 20
        m.Uid = "1"
        result.Matches = append(result.Matches, m)
        m = Match{}
        m.Start = -1
        m.End = -1
        m.Uid = "2"
        result.Matches = append(result.Matches, m)
        m = Match{}
        m.Start = -1
        m.End = -1
        m.Uid = "3"
        result.Matches = append(result.Matches, m)
        m = Match{}
        m.Start = 40
        m.End = 50
        m.Uid = "4"
        result.Matches = append(result.Matches, m)
        min_pos, pre_spacer, max_pos, post_spacer := result.FindSurroundingPositions("3")
        if min_pos != 20 || pre_spacer == false || max_pos != 40 || post_spacer == true {
            t.Errorf("Invalid result: %d %t %d %t", min_pos, pre_spacer, max_pos, post_spacer)
        }
}

func TestFindSurroundingPositionsWithPostUnknowns(t *testing.T) {
        result := NewResult()
        m := Match{}
        m.Start = 10
        m.End = 20
        m.Uid = "1"
        result.Matches = append(result.Matches, m)
        m = Match{}
        m.Start = 20
        m.End = 30
        m.Uid = "2"
        result.Matches = append(result.Matches, m)
        m = Match{}
        m.Start = -1
        m.End = -1
        m.Uid = "3"
        result.Matches = append(result.Matches, m)
        m = Match{}
        m.Start = -1
        m.End = -1
        m.Uid = "4"
        result.Matches = append(result.Matches, m)
        m = Match{}
        m.Start = 50
        m.End = 60
        m.Uid = "5"
        result.Matches = append(result.Matches, m)
        min_pos, pre_spacer, max_pos, post_spacer := result.FindSurroundingPositions("3")
        if min_pos != 30 || pre_spacer == true || max_pos != 50 || post_spacer == false {
            t.Errorf("Invalid result: %d %t %d %t", min_pos, pre_spacer, max_pos, post_spacer)
        }
}
