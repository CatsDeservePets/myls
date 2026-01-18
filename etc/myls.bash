# Bash completion for myls.
#
# Save this file as `myls` in a directory used by bash completions.

_myls() {
	local cur="${COMP_WORDS[COMP_CWORD]}"
	local prev="${COMP_WORDS[COMP_CWORD - 1]}"

	local -a opts=(
		-h -help
		-V -version
		-a
		-d
		-l
		-r
		-1
		-dirsfirst
		-git
		-sort
	)

	if [[ "$prev" == "-sort" ]]; then
		COMPREPLY=($(compgen -W "name extension size time git" -- "$cur"))
	elif [[ "$cur" == -* ]]; then
		COMPREPLY=($(compgen -W "${opts[*]}" -- "$cur"))
	else
		COMPREPLY=($(compgen -f -d -- "$cur"))
	fi
}

complete -o filenames -F _myls myls
