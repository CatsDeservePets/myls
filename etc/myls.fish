# Fish completion for myls.
#
# Save this file as `myls.fish` in a directory used by fish completions (`$fish_complete_path`).

complete -c myls -o h -o help -d 'show help message and exit'
complete -c myls -o V -o version -d 'show program\'s version number and exit'
complete -c myls -o a -d 'do not ignore entries starting with .'
complete -c myls -o d -d 'list directories themselves, not their contents'
complete -c myls -o l -d 'use a long listing format'
complete -c myls -o r -d 'reverse order while sorting'
complete -c myls -o 1 -d 'display one entry per line'
complete -c myls -o dirsfirst -d 'show directories above regular files'
complete -c myls -o git -d 'display git status'
complete -c myls -o sort -x -k -a "name\t extension\t size\t time\t git\t" -d 'one of: name, extension, size, time, git (default: name)'
