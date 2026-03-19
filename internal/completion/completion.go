// Package completion generates shell completion scripts for bash, zsh, and fish.
package completion

// Bash returns a bash completion script for crontui.
func Bash() string {
	return `_crontui() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local cmds="list ls add delete rm enable disable validate preview runnow run backup restore export import help version"
    COMPREPLY=($(compgen -W "${cmds}" -- "${cur}"))
}
complete -F _crontui crontui
`
}

// Zsh returns a zsh completion script for crontui.
func Zsh() string {
	return `#compdef crontui

_crontui() {
    local -a commands
    commands=(
        'list:List all cron jobs'
        'ls:List all cron jobs (alias)'
        'add:Add a new cron job'
        'delete:Delete a cron job'
        'rm:Delete a cron job (alias)'
        'enable:Enable a cron job'
        'disable:Disable a cron job'
        'validate:Validate a cron expression'
        'preview:Preview next runs'
        'runnow:Execute a job immediately'
        'run:Execute a job immediately (alias)'
        'backup:Create a crontab backup'
        'restore:Restore from backup'
        'export:Export jobs'
        'import:Import jobs from JSON'
        'help:Show help'
        'version:Show version'
    )
    _describe 'command' commands
}

_crontui "$@"
`
}

// Fish returns a fish completion script for crontui.
func Fish() string {
	return `complete -c crontui -f
complete -c crontui -n '__fish_use_subcommand' -a list -d 'List all cron jobs'
complete -c crontui -n '__fish_use_subcommand' -a ls -d 'List all cron jobs (alias)'
complete -c crontui -n '__fish_use_subcommand' -a add -d 'Add a new cron job'
complete -c crontui -n '__fish_use_subcommand' -a delete -d 'Delete a cron job'
complete -c crontui -n '__fish_use_subcommand' -a rm -d 'Delete a cron job (alias)'
complete -c crontui -n '__fish_use_subcommand' -a enable -d 'Enable a cron job'
complete -c crontui -n '__fish_use_subcommand' -a disable -d 'Disable a cron job'
complete -c crontui -n '__fish_use_subcommand' -a validate -d 'Validate a cron expression'
complete -c crontui -n '__fish_use_subcommand' -a preview -d 'Preview next runs'
complete -c crontui -n '__fish_use_subcommand' -a runnow -d 'Execute a job immediately'
complete -c crontui -n '__fish_use_subcommand' -a run -d 'Execute a job immediately (alias)'
complete -c crontui -n '__fish_use_subcommand' -a backup -d 'Create a crontab backup'
complete -c crontui -n '__fish_use_subcommand' -a restore -d 'Restore from backup'
complete -c crontui -n '__fish_use_subcommand' -a export -d 'Export jobs'
complete -c crontui -n '__fish_use_subcommand' -a import -d 'Import jobs from JSON'
complete -c crontui -n '__fish_use_subcommand' -a help -d 'Show help'
complete -c crontui -n '__fish_use_subcommand' -a version -d 'Show version'
`
}
