This repo contains some handy tools for working with JSON and JSON schemas.

## check-json-schemas -dir ./schemas

This command validates that all the JSON files in a directory contain a
schema that github.com/xeipuuv/gojsonschema can load and validate documents
against.

-dir string
    The directory containing the schemas to validate.
-help
    Show usage information.
-verbose
    Print verbose output.

## json-ordered-tidy ... ./file.json ./dir

This command tidies all the files it finds, preserving the existing map key
order. If given a directory it looks for files matching the given -ext
value.

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
