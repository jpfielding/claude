# tally

tally is a tool. it counts things.

## Install
go install

## Usage
tally [flags] files

flags:
-w words
-l lines
-c chars
-j json
-u unique (only with -w)

also you can pipe stuff in. If no files it reads stdin. exit code 2 on error, 1 if any file missing but others ok.

## notes
config file is ~/.tallyrc, same flags one per line. env TALLY_NOCOLOR disables color. ignores binary files unless --binary.
