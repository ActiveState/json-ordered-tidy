The `json-ordered-tidy` command tidies all the files it finds, preserving the
existing map key order. If given a directory it looks for files matching the
given -ext value.

-check
    Run in check mode. In this mode we exit 0 if all files are already tidy, otherwise the exit status is 1.
-ext string
    The file extension to match against. Only files with this extension will be tidied. (default ".json")
-help
    Show usage information.
-indent string
    The string with which to indent JSON. Defaults to 4 spaces. (default "    ")
-verbose
    Be more verbose with output.
