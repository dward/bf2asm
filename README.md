# bf2asm
Brainfuck to amd64 assembly compiler

The compiler is written in Go (http://www.golang.com/). 
This is an experiment to parse brainfuck into an AST, optimize, and then dump out as amd64.

Usage:
go build

./bf2asm <file.b> > run.asm
nasm -f elf64 -o run.o run.asm
ld run.o -o run
