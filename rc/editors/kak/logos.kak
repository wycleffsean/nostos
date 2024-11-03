# Detection
# ‾‾‾‾‾‾‾‾‾

hook global BufCreate .*[.](lo) %{
    set-option buffer filetype logos
}

# Initialization
# ‾‾‾‾‾‾‾‾‾‾‾‾‾‾

hook global WinSetOption filetype=logos %{
    require-module logos

    set-option window static_words %opt{logos_static_words}

    hook window ModeChange pop:insert:.* -group logos-trim-indent logos-trim-indent
    hook window InsertChar \n -group logos-insert logos-insert-on-new-line
    hook window InsertChar \n -group logos-indent logos-indent-on-new-line
    hook -once -always window WinSetOption filetype=.* %{ remove-hooks window logos-.+ }
}

hook -group logos-highlight global WinSetOption filetype=logos %{
    add-highlighter window/logos ref logos
    hook -once -always window WinSetOption filetype=.* %{ remove-highlighter window/logos }
}


provide-module logos %{

# Highlighters
# ‾‾‾‾‾‾‾‾‾‾‾‾

add-highlighter shared/logos regions
add-highlighter shared/logos/code      default-region group
add-highlighter shared/logos/double_string region '"' (?<!\\)(\\\\)*"       fill string
add-highlighter shared/logos/single_string region "'" "'"                   fill string
add-highlighter shared/logos/comment       region '(?:^| )#' $              fill comment

add-highlighter shared/logos/code/ regex ^(---|\.\.\.)$ 0:meta
add-highlighter shared/logos/code/ regex ^(\h*:\w*) 0:keyword
add-highlighter shared/logos/code/ regex \b(true|false|null)\b 0:value
add-highlighter shared/logos/code/ regex ^\h*-?\h*(\S+): 1:attribute

evaluate-commands %sh{
    # Grammar
    keywords="let"
    keywords="${keywords}|ensure|false|for|if|in|module|next|nil|not|or|private|protected|public|redo"
    keywords="${keywords}|rescue|retry|return|self|super|then|true|undef|unless|until|when|while|yield"
    attributes="attr_reader|attr_writer|attr_accessor"
    values="false|true|nil"
    meta="require|require_relative|include|extend"

    # Add the language's grammar to the static completion list
    printf %s\\n "declare-option str-list logos_static_words ${keywords} ${attributes} ${values} ${meta}" | tr '|' ' '

    # Highlight keywords
    printf %s "
        add-highlighter shared/logos/code/ regex \b(${keywords})[^0-9A-Za-z_!?] 1:keyword
        add-highlighter shared/logos/code/ regex \b(${attributes})\b 0:attribute
        add-highlighter shared/logos/code/ regex \b(${values})\b 0:value
        add-highlighter shared/logos/code/ regex \b(${meta})\b 0:meta
    "
}

# Commands
# ‾‾‾‾‾‾‾‾

define-command -hidden logos-trim-indent %{
    # remove trailing white spaces
    try %{ execute-keys -draft -itersel x s \h+$ <ret> d }
}

define-command -hidden logos-insert-on-new-line %{
    evaluate-commands -draft -itersel %{
        # copy '#' comment prefix and following white spaces
        try %{ execute-keys -draft k x s ^\h*\K#\h* <ret> y gh j P }
    }
}

define-command -hidden logos-indent-on-new-line %{
    evaluate-commands -draft -itersel %{
        # preserve previous line indent
        try %{ execute-keys -draft <semicolon> K <a-&> }
        # filter previous line
        try %{ execute-keys -draft k : logos-trim-indent <ret> }
        # indent after :
        try %{ execute-keys -draft , k x <a-k> :$ <ret> j <a-gt> }
        # indent after -
        try %{ execute-keys -draft , k x <a-k> ^\h*- <ret> j <a-gt> }
    }
}

}
