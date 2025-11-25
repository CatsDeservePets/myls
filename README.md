# myls - My interpretation of the ls(1) command

`myls` is a highly opinionated `ls` clone tailored to my own workflow.
It does not aim to be compatible with either BSD or GNU `ls`.

## Installation

```shell
go install github.com/CatsDeservePets/myls@latest
```

## Usage

```
usage: myls [-h] [-a] [-l] [-r] [-dirsfirst] [-sort WORD] [file ...]

positional arguments:
  file        files or directories to display

options:
  -h, -help   show this help message and exit
  -a          do not ignore entries starting with .
  -l          use a long listing format
  -r          reverse order while sorting
  -dirsfirst  show directories above regular files
  -sort WORD  one of: name, extension, size, time (default: name)
```
