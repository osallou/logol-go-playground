package logol


import (
        "testing"
        logol "github.com/osallou/logol-go-playground/lib/types"
        //"log"
)

func TestEvaluate(t *testing.T){
    contextVars := make(map[string]logol.Match)

    m1 := logol.NewMatch()
    m1.Id = "R1"
    m1.Start = 1
    m1.End = 2
    contextVars["R1"] = m1
    m2 := logol.NewMatch()
    m2.Id = "R2"
    m2.Start = 4
    m2.End = 6
    contextVars["R2"] = m2
    res := Evaluate("@R1 < @R2", contextVars)
    if ! res {
        t.Errorf("Invalid, @R1 < @R2: %t", res)
    }
    res = Evaluate("@R2 < @R1", contextVars)
    if res {
        t.Errorf("Invalid, @R2 < @R1: %t", res)
    }

}

func TestPercentMatch(t *testing.T) {
    res, percent := CheckAlphabetPercent("aaaaaaaaat", "a", 80)
    if ! res {
        t.Errorf("Invalid, %t: %d", res, percent)
    }
    res, percent = CheckAlphabetPercent("aaaaaaaaat", "t", 80)
    if res {
        t.Errorf("Invalid, %t: %d", res, percent)
    }
    res, percent = CheckAlphabetPercent("aaaaaaccct", "ac", 80)
    if ! res {
        t.Errorf("Invalid, %t: %d", res, percent)
    }
    res, percent = CheckAlphabetPercent("aaaaaaccct", "tg", 80)
    if res {
        t.Errorf("Invalid, %t: %d", res, percent)
    }
}

func TestUndefinedVars(t *testing.T) {
    contextVars := make(map[string]logol.Match)
    //contextVars["R1"] = logol.NewMatch()
    hasUndefined, undefinedVars := HasUndefinedRangeVars("12", contextVars)
    if hasUndefined {
        t.Errorf("Invalid, %t: %s", hasUndefined, undefinedVars)
    }

    m1 := logol.NewMatch()
    m1.Id = "R1"
    m1.Start = 1
    m1.End = 2
    contextVars["R1"] = m1
    m2 := logol.NewMatch()
    m2.Id = "R1"
    m2.Start = 1
    m2.End = 2
    contextVars["R2"] = m2
    hasUndefined, undefinedVars = HasUndefinedRangeVars("@@R1", contextVars)
    if hasUndefined {
        t.Errorf("Invalid, %t: %s", hasUndefined, undefinedVars)
    }
    hasUndefined, undefinedVars = HasUndefinedRangeVars("@@R3", contextVars)
    if ! hasUndefined {
        t.Errorf("Invalid, %t: %s", hasUndefined, undefinedVars)
    }
    hasUndefined, undefinedVars = HasUndefinedRangeVars("@R1 + @R2", contextVars)
    if hasUndefined {
        t.Errorf("Invalid, %t: %s", hasUndefined, undefinedVars)
    }
    hasUndefined, undefinedVars = HasUndefinedRangeVars("@R1 + @R3", contextVars)
    if ! hasUndefined {
        t.Errorf("Invalid, %t: %s", hasUndefined, undefinedVars)
    }else {
        if undefinedVars[0] != "R3" {
            t.Errorf("Invalid, %t: %s", hasUndefined, undefinedVars)
        }
    }
    hasUndefined, undefinedVars = HasUndefinedRangeVars("?R1", contextVars)
    if hasUndefined {
        t.Errorf("Invalid, %t: %s", hasUndefined, undefinedVars)
    }
    hasUndefined, undefinedVars = HasUndefinedRangeVars("?R3", contextVars)
    if ! hasUndefined {
        t.Errorf("Invalid, %t: %s", hasUndefined, undefinedVars)
    }
}
