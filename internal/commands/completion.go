package commands

import (
	"bytes"
	"fmt"
	"io"

	"github.com/docker/cli/cli"
	"github.com/docker/cli/cli/command"
	"github.com/spf13/cobra"
)

func completionCmd(dockerCli command.Cli, rootCmd *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "completion SHELL",
		Short: "Generates completion scripts for the specified shell (bash or zsh)",
		Long: `# Load the docker-app completion code for bash into the current shell
. <(docker-app completion bash)
# Set the docker-app completion code for bash to autoload on startup in your ~/.bashrc,
# ~/.profile or ~/.bash_profile
. <(docker-app completion bash)
# Note: bash-completion is needed.

# Load the docker-app completion code for zsh into the current shell
source <(docker-app completion zsh)
# Set the docker-app completion code for zsh to autoload on startup in your ~/.zshrc
source <(docker-app completion zsh)
`,
		Args: cli.RequiresMaxArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case len(args) == 0:
				return rootCmd.GenBashCompletion(dockerCli.Out())
			case args[0] == "bash":
				return rootCmd.GenBashCompletion(dockerCli.Out())
			case args[0] == "zsh":
				return runCompletionZsh(dockerCli.Out(), rootCmd)
			default:
				return fmt.Errorf("%q is not a supported shell", args[0])
			}
		},
	}
}

const (
	// Largely inspired by https://github.com/kubernetes/kubernetes/blob/master/pkg/kubectl/cmd/completion.go
	zshHead = `#compdef dockerapp
__dockerapp_bash_source() {
	alias shopt=':'
	alias _expand=_bash_expand
	alias _complete=_bash_comp
	emulate -L sh
	setopt kshglob noshglob braceexpand
 	source "$@"
}
 __dockerapp_type() {
	# -t is not supported by zsh
	if [ "$1" == "-t" ]; then
		shift
 		# fake Bash 4 to disable "complete -o nospace". Instead
		# "compopt +-o nospace" is used in the code to toggle trailing
		# spaces. We don't support that, but leave trailing spaces on
		# all the time
		if [ "$1" = "__dockerapp_compopt" ]; then
			echo builtin
			return 0
		fi
	fi
	type "$@"
}
 __dockerapp_compgen() {
	local completions w
	completions=( $(compgen "$@") ) || return $?
 	# filter by given word as prefix
	while [[ "$1" = -* && "$1" != -- ]]; do
		shift
		shift
	done
	if [[ "$1" == -- ]]; then
		shift
	fi
	for w in "${completions[@]}"; do
		if [[ "${w}" = "$1"* ]]; then
			echo "${w}"
		fi
	done
}
 __dockerapp_compopt() {
	true # don't do anything. Not supported by bashcompinit in zsh
}
 __dockerapp_ltrim_colon_completions()
{
	if [[ "$1" == *:* && "$COMP_WORDBREAKS" == *:* ]]; then
		# Remove colon-word prefix from COMPREPLY items
		local colon_word=${1%${1##*:}}
		local i=${#COMPREPLY[*]}
		while [[ $((--i)) -ge 0 ]]; do
			COMPREPLY[$i]=${COMPREPLY[$i]#"$colon_word"}
		done
	fi
}
 __dockerapp_get_comp_words_by_ref() {
	cur="${COMP_WORDS[COMP_CWORD]}"
	prev="${COMP_WORDS[${COMP_CWORD}-1]}"
	words=("${COMP_WORDS[@]}")
	cword=("${COMP_CWORD[@]}")
}
 __dockerapp_filedir() {
	local RET OLD_IFS w qw
 	__debug "_filedir $@ cur=$cur"
	if [[ "$1" = \~* ]]; then
		# somehow does not work. Maybe, zsh does not call this at all
		eval echo "$1"
		return 0
	fi
 	OLD_IFS="$IFS"
	IFS=$'\n'
	if [ "$1" = "-d" ]; then
		shift
		RET=( $(compgen -d) )
	else
		RET=( $(compgen -f) )
	fi
	IFS="$OLD_IFS"
 	IFS="," __debug "RET=${RET[@]} len=${#RET[@]}"
 	for w in ${RET[@]}; do
		if [[ ! "${w}" = "${cur}"* ]]; then
			continue
		fi
		if eval "[[ \"\${w}\" = *.$1 || -d \"\${w}\" ]]"; then
			qw="$(__dockerapp_quote "${w}")"
			if [ -d "${w}" ]; then
				COMPREPLY+=("${qw}/")
			else
				COMPREPLY+=("${qw}")
			fi
		fi
	done
}
 __dockerapp_quote() {
    if [[ $1 == \'* || $1 == \"* ]]; then
        # Leave out first character
        printf %q "${1:1}"
    else
    	printf %q "$1"
    fi
}
 autoload -U +X bashcompinit && bashcompinit
 # use word boundary patterns for BSD or GNU sed
LWORD='[[:<:]]'
RWORD='[[:>:]]'
if sed --help 2>&1 | grep -q GNU; then
	LWORD='\<'
	RWORD='\>'
fi
 __dockerapp_convert_bash_to_zsh() {
	sed \
	-e 's/declare -F/whence -w/' \
	-e 's/_get_comp_words_by_ref "\$@"/_get_comp_words_by_ref "\$*"/' \
	-e 's/local \([a-zA-Z0-9_]*\)=/local \1; \1=/' \
	-e 's/flags+=("\(--.*\)=")/flags+=("\1"); two_word_flags+=("\1")/' \
	-e 's/must_have_one_flag+=("\(--.*\)=")/must_have_one_flag+=("\1")/' \
	-e "s/${LWORD}_filedir${RWORD}/__dockerapp_filedir/g" \
	-e "s/${LWORD}_get_comp_words_by_ref${RWORD}/__dockerapp_get_comp_words_by_ref/g" \
	-e "s/${LWORD}__ltrim_colon_completions${RWORD}/__dockerapp_ltrim_colon_completions/g" \
	-e "s/${LWORD}compgen${RWORD}/__dockerapp_compgen/g" \
	-e "s/${LWORD}compopt${RWORD}/__dockerapp_compopt/g" \
	-e "s/${LWORD}declare${RWORD}/builtin declare/g" \
	-e "s/\\\$(type${RWORD}/\$(__dockerapp_type/g" \
	<<'BASH_COMPLETION_EOF'
`
	zshTail = `
BASH_COMPLETION_EOF
}
 __dockerapp_bash_source <(__dockerapp_convert_bash_to_zsh)
_complete dockerapp 2>/dev/null
`
)

func runCompletionZsh(out io.Writer, rootCmd *cobra.Command) error {
	fmt.Fprint(out, zshHead)
	buf := new(bytes.Buffer)
	rootCmd.GenBashCompletion(buf)
	fmt.Fprint(out, buf.String())
	fmt.Fprint(out, zshTail)
	return nil
}
