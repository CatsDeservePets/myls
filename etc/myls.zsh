#compdef myls

# Zsh completion for myls.
#
# Save this file as `_myls` in a directory used by zsh completions (`$fpath`) and ensure `compinit` is enabled.

_arguments -s \
	'(-h -help)'{-h,-help}'[show help message and exit]' \
	'(-V -version)'{-V,-version}"[show program's version number and exit]" \
	'-a[do not ignore entries starting with .]' \
	'-d[list directories themselves, not their contents]' \
	'-l[use a long listing format]' \
	'-r[reverse order while sorting]' \
	'-1[display one entry per line]' \
	'-dirsfirst[show directories above regular files]' \
	'-git[display git status]' \
	'-sort[one of: name, extension, size, time, git (default: name)]:sort:(name extension size time git)' \
	'*:file:_files'
