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

## Example output

```
$ ./myls -l -a
drwxr-xr-x    8 Nov 25 17:42 ./
drwxr-xr-x   31 Nov 24 00:28 ../
drwxr-xr-x   14 Nov 25 17:42 .git/
-rw-r--r--   5B Nov 20 22:11 .gitignore
-rw-r--r--  50B Nov 20 22:11 go.mod
-rw-r--r-- 7.4K Nov 25 17:29 main.go
-rw-r--r-- 1.3K Nov 23 23:00 misc.go
-rw-r--r-- 1.6K Nov 23 23:00 misc_windows.go
-rwxr-xr-x 2.5M Nov 25 17:27 myls*
-rw-r--r-- 727B Nov 25 17:38 README.md
```

```
C:\Programming\myls> .\myls.exe -l -a
d----    8 Nov 25 17:42 .\
d----   31 Nov 24 00:28 ..\
d--h-   14 Nov 25 17:42 .git\
-a---   5B Nov 20 22:11 .gitignore
-a---  50B Nov 20 22:11 go.mod
-a--- 7.4K Nov 25 17:29 main.go
-a--- 1.3K Nov 23 23:00 misc.go
-a--- 1.6K Nov 23 23:00 misc_windows.go
-a--- 2.5M Nov 25 17:27 myls.exe*
-a--- 727B Nov 25 17:38 README.md
```
