package main


import (
        "fmt"
        //"log"
        //"gopkg.in/yaml.v2"
        "github.com/streadway/amqp"
        //logol "org.irisa.genouest/logol/lib/types"
        msgHandler "org.irisa.genouest/logol/lib/listener"
)

var grammar = `
morphisms:
  - foo:
    - a
    - g
models:
  mod2:
    comment: 'mod2(+R2)'
    start: 'var1'
    param:
      - R2
    vars:
        var1:
            value: null
            string_constraints:
                content: 'R2'
            next: null


  mod1:
   comment: 'mod1(-R1)'
   param:
     - 'R1'
   start: 'var1'
   vars:
     var1:
         value: 'cc'
         next:
           - var2
     var2:
         value: 'aaa'
         string_constraints:
             save_as: 'R1'
         next:
          - var3
          - var4
     var3:
         value: null
         string_constraints:
           content: 'R1'
         next:
           - var5
     var4:
         comments: 'mod2(+R1)'
         value: null
         model:
             name: 'mod2'
             param:
               - R1
         next:
           - var5
     var5:
         value: 'cgt'
         next: null

run:
 - mod1
`


// Note: struct fields must be public in order for unmarshal to
// correctly populate the data.
type T struct {
        A string
        B struct {
                RenamedC int   `yaml:"c"`
                D        []int `yaml:",flow"`
        }
}



func main() {
    /*
    err, t := logol.LoadGrammar([]byte(grammar))

    if err != nil {
            log.Fatalf("error: %v", err)
    }

    fmt.Printf("--- t:\n%v\n\n", t)
    d, err := t.DumpGrammar()

    if err != nil {
            log.Fatalf("error: %v", err)
    }
    fmt.Printf("--- t dump:\n%s\n\n", string(d))

    match := logol.NewMatch()
    m, err := match.Dumps()
    if err != nil {
            log.Fatalf("error: %v", err)
    }
    fmt.Printf("--- match dump:\n%s\n\n", string(m))

    result := logol.NewResult()
    result.Matches = append(result.Matches, match)
    result.Context = append(result.Context, []logol.Match{match})
    contextVar := map[string]logol.Match{"MYVAR": match}
    result.ContextVars = append(result.ContextVars, contextVar)
    r, err := result.Dumps()
    if err != nil {
            log.Fatalf("error: %v", err)
    }
    fmt.Printf("--- result dump:\n%s\n\n", string(r))
    */

    connUrl := fmt.Sprintf("amqp://%s:%s@%s:%d/",
        "guest", "guest", "localhost", 5672)
    conn, _ := amqp.Dial(connUrl)
    ch, _ := conn.Channel()
    _, _ = ch.QueueDeclare(
      "logol-analyse-test", // name
      false,   // durable
      false,   // delete when usused
      false,   // exclusive
      false,   // no-wait
      nil,     // arguments
    )

    handler := msgHandler.NewMsgHandler("localhost", 5672, "guest", "guest")
    handler.Listen("test", nil)
}
