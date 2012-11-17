scan
====

A generic scanner based on the scanner in http://golang.org/pkg/text/template/parse/.

Package scan contains the generic parts of Rob Pike's scanner for the
text/template package in Go's standard library. Only a few changes
were made to turn the code into a library. 

The package can be used to write scanners by supplying the missing
parts: a list of tokens and state functions implementing the state
machine. 

See the test code for a simple example and the original lexer in
http://go.googlecode.com/hg/src/pkg/text/template/parser/lex.go
for a real-world example.
