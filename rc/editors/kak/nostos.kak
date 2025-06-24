# Detection
# ‾‾‾‾‾‾‾‾‾

hook global BufCreate .*[.](no) %{
    set-option buffer filetype nostos
}

# Initialization
# ‾‾‾‾‾‾‾‾‾‾‾‾‾‾

hook global WinSetOption filetype=nostos %{
    require-module nostos

    hook window ModeChange pop:insert:.* -group nostos-trim-indent nostos-trim-indent
    hook window InsertChar \n -group nostos-insert nostos-insert-on-new-line
    hook window InsertChar \n -group nostos-indent nostos-indent-on-new-line
    hook -once -always window WinSetOption filetype=.* %{ remove-hooks window nostos-.+ }
}

hook -group nostos-highlight global WinSetOption filetype=nostos %{
    add-highlighter window/nostos ref nostos
    hook -once -always window WinSetOption filetype=.* %{ remove-highlighter window/nostos }
}

provide-module nostos %{

# Highlighters
# ‾‾‾‾‾‾‾‾‾‾‾‾

add-highlighter shared/nostos regions
add-highlighter shared/nostos/code      default-region group
add-highlighter shared/nostos/double_string region '"' (?<!\\)(\\\\)*"       fill string
add-highlighter shared/nostos/single_string region "'" "'"                   fill string
add-highlighter shared/nostos/comment       region '(?:^| )#' $              fill comment

add-highlighter shared/nostos/code/ regex ^(---|\.\.\.)$ 0:meta
add-highlighter shared/nostos/code/ regex ^(\h*:\w*) 0:keyword
add-highlighter shared/nostos/code/ regex \b(true|false|null)\b 0:value
add-highlighter shared/nostos/code/ regex ^\h*-?\h*(\S+): 1:attribute
add-highlighter shared/nostos/code/ regex => 0:keyword

# Commands
# ‾‾‾‾‾‾‾‾

define-command -hidden nostos-trim-indent %{
    # remove trailing white spaces
    try %{ execute-keys -draft -itersel x s \h+$ <ret> d }
}

define-command -hidden nostos-insert-on-new-line %{
    evaluate-commands -draft -itersel %{
        # copy '#' comment prefix and following white spaces
        try %{ execute-keys -draft k x s ^\h*\K#\h* <ret> y gh j P }
    }
}

define-command -hidden nostos-indent-on-new-line %{
    evaluate-commands -draft -itersel %{
        # preserve previous line indent
        try %{ execute-keys -draft <semicolon> K <a-&> }
        # filter previous line
        try %{ execute-keys -draft k : nostos-trim-indent <ret> }
        # indent after :
        try %{ execute-keys -draft , k x <a-k> :$ <ret> j <a-gt> }
        # indent after -
        try %{ execute-keys -draft , k x <a-k> ^\h*- <ret> j <a-gt> }
    }
}

}
