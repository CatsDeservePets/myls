# PowerShell completion for myls.
#
# Save this file as `myls.ps1` anywhere you like and dot-source it from your PowerShell profile (`$PROFILE`).

using namespace System.Management.Automation

Register-ArgumentCompleter -Native -CommandName 'myls' -ScriptBlock {
	param($wordToComplete, $commandAst, $cursorPosition)

	$sortValues = @('name', 'extension', 'size', 'time', 'git')

	$completions = @(
		[CompletionResult]::new('-h',         '-h',         [CompletionResultType]::ParameterName, 'show help message and exit')
		[CompletionResult]::new('-help',      '-help',      [CompletionResultType]::ParameterName, 'show help message and exit')
		[CompletionResult]::new('-V',         '-V',         [CompletionResultType]::ParameterName, "show program's version number and exit")
		[CompletionResult]::new('-version',   '-version',   [CompletionResultType]::ParameterName, "show program's version number and exit")
		[CompletionResult]::new('-a',         '-a',         [CompletionResultType]::ParameterName, 'do not ignore entries starting with .')
		[CompletionResult]::new('-d',         '-d',         [CompletionResultType]::ParameterName, 'list directories themselves, not their contents')
		[CompletionResult]::new('-l',         '-l',         [CompletionResultType]::ParameterName, 'use a long listing format')
		[CompletionResult]::new('-r',         '-r',         [CompletionResultType]::ParameterName, 'reverse order while sorting')
		[CompletionResult]::new('-1',         '-1',         [CompletionResultType]::ParameterName, 'display one entry per line')
		[CompletionResult]::new('-dirsfirst', '-dirsfirst', [CompletionResultType]::ParameterName, 'show directories above regular files')
		[CompletionResult]::new('-git',       '-git',       [CompletionResultType]::ParameterName, 'display git status')
		[CompletionResult]::new('-sort ',     '-sort',      [CompletionResultType]::ParameterName, 'one of: name, extension, size, time, git (default: name)')
	)

	if ($wordToComplete.StartsWith('-')) {
		$completions.Where{ $_.CompletionText -like "$wordToComplete*" } |
			Sort-Object -Property ListItemText
		return
	}

	$previousElement = $commandAst.CommandElements |
		Where-Object { $_.Extent.EndOffset -lt $cursorPosition } |
		Select-Object -Last 1

	if ($previousElement.Extent.Text -eq '-sort') {
		$sortValues.Where{ $_ -like "$wordToComplete*" } |
			ForEach-Object {
				[CompletionResult]::new($_, $_, [CompletionResultType]::ParameterValue, $_)
			}
	}
}
