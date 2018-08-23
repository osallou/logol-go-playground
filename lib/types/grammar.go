package logol

import (
        "gopkg.in/yaml.v2"
        "strings"
        "strconv"
        "log"
        logs "org.irisa.genouest/logol/lib/log"
)

var logger = logs.GetLogger("logol.types")

type Morphism struct {
    Morph map[string][]string
}
type ParamDef struct {
    Inputs []string
    Outputs []string
}
type VariableModel struct {
    Name string
    Param []string
    RepeatMin int
    RepeatMax int
}
type RangeConstraint struct {
    Min string
    Max string
}
type StringConstraint struct {
    Content string
    SaveAs string
    Size RangeConstraint
    Start RangeConstraint
    End RangeConstraint
    Morphism string // name of morphism
    Reverse bool
}
type StructConstraint struct {
    Cost RangeConstraint // range [3,4]
    Distance RangeConstraint // range [10,20]
    Percent string // ac:80 80% of ac
}
// Define additional constraint on match
type NegConstraint struct {
    Value string
    String_constraints StringConstraint
    Struct_constraints StructConstraint
}
type Variable struct {
    Value string
    Next []string
    Model VariableModel
    String_constraints StringConstraint
    Struct_constraints StructConstraint
    Negative_constraints[] NegConstraint
    Overlap bool
}
type MetaConstraint struct {
    Vars []string // list of variables
    String_constraints []StringConstraint
    Struct_constraints []StructConstraint
}
type Model struct {
    Comment string
    Param []string
    Meta []MetaConstraint
    Start[] string
    Vars map[string]Variable
}

type MainModel struct {
    Model string
    Param []string
}

type Grammar struct {
    Morphisms map[string]Morphism
    Models map[string]Model
    Run []MainModel
    Sequence string
}

func LoadGrammar(grammar []byte) (error, Grammar) {
    g := Grammar{}
    err := yaml.Unmarshal(grammar, &g)
    return err, g
}

func (g Grammar) DumpGrammar() (out []byte, err error) {
    return yaml.Marshal(&g)
}

func (lvar Variable) HasSizeConstraint()  bool {
    if lvar.String_constraints.Size.Min == "" && lvar.String_constraints.Size.Max == "" {
            return false
    }
    return true
}
func (lvar Variable) GetSizeConstraint() (min string, max string) {
    return lvar.String_constraints.Size.Min, lvar.String_constraints.Size.Max
}

func (lvar Variable) HasStartConstraint()  bool {
    if lvar.String_constraints.Start.Min == "" && lvar.String_constraints.Start.Max == "" {
            return false
    }
    return true
}
func (lvar Variable) GetStartConstraint() (min string, max string) {
    return lvar.String_constraints.Start.Min, lvar.String_constraints.Start.Max
}

func (lvar Variable) HasEndConstraint()  bool {
    if lvar.String_constraints.End.Min == "" && lvar.String_constraints.End.Max == "" {
            return false
    }
    return true
}
func (lvar Variable) GetEndConstraint() (min string, max string) {
    return lvar.String_constraints.End.Min, lvar.String_constraints.End.Max
}

func (lvar Variable) HasCostConstraint()  bool {
    if lvar.Struct_constraints.Cost.Min == "" && lvar.Struct_constraints.Cost.Max == "" {
            return false
    }
    return true
}
func (lvar Variable) GetCostConstraint() (min string, max string) {
    return lvar.Struct_constraints.Cost.Min, lvar.Struct_constraints.Cost.Max
}
func (lvar Variable) HasDistanceConstraint()  bool {
    if lvar.Struct_constraints.Distance.Min == "" && lvar.Struct_constraints.Distance.Max == "" {
            return false
    }
    return true
}
func (lvar Variable) GetDistanceConstraint() (min string, max string) {
    return lvar.Struct_constraints.Distance.Min, lvar.Struct_constraints.Distance.Max
}
func (lvar Variable) HasPercentConstraint()  bool {
    if lvar.Struct_constraints.Percent == "" {
            return false
    }
    return true
}
func (lvar Variable) GetPercentConstraint() (alphabet string, percent int, error bool) {
    elts := strings.Split(lvar.Struct_constraints.Percent, ":")
    if len(elts) != 2 {
        log.Printf("Invalid percent constraint skipping it")
        return "", -1, true
    }
    percent, err := strconv.Atoi(elts[1])
    if err != nil {
        log.Printf("Invalid percent constraint skipping it")
        return "", -1, true
    }
    return elts[0], percent, false
}

func(lvar Variable) HasContentConstraint() bool {
    if lvar.Value != "" || lvar.String_constraints.Content != "" {
        return true
    }
    return false
}
// Return content content string
func(lvar Variable) GetContentConstraint() (content string, isFixed bool, err bool) {
    if lvar.Value != "" {
        return lvar.Value, true, false
    }
    if lvar.String_constraints.Content != "" {
        return lvar.String_constraints.Content, false, false
    }
    return "", false, true
}
func(lvar Variable) HasReverseConstraint() bool {
    return  lvar.String_constraints.Reverse
}
func(lvar Variable) HasMorphism() bool {
    return  lvar.String_constraints.Morphism != ""
}
func(lvar Variable) GetMorphism(morphisms map[string]Morphism) Morphism {
    morphName := lvar.String_constraints.Morphism
    log.Printf("Get morphism variable %s", morphName)
    morph, ok := morphisms[morphName]
    if ! ok {
        if morphName == "wc" {
            morph = Morphism{}
            morph.Morph = make(map[string][]string)
            morph.Morph["a"] = make([]string, 1)
            morph.Morph["a"][0] = "t"
            morph.Morph["c"] = make([]string, 1)
            morph.Morph["c"][0] = "g"
            morph.Morph["g"] = make([]string, 1)
            morph.Morph["g"][0] = "c"
            morph.Morph["t"] = make([]string, 1)
            morph.Morph["t"][0] = "a"

        }
    }
    return morph
}
