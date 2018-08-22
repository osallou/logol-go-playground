# about

Logol rewriting in go

# status

in development

# deps

rabbitmq and redis

Needs cassiopee >= 1.0.9, org.irisa.genouest/cassiopee is built upon this version

# running


in cmd/logol[]/
1 or more : logolMessage
1: logolResult
1: logolCassie

then logolClient

testmsg will send grammar and sequence info , once search is over it will stop processes
