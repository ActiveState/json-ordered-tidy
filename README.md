## `json-ordered-tidy ... ./file.json ./dir`

This command tidies all the files it finds, preserving the existing map key
order. If given a directory it looks for files matching the given -ext
value.

This tidier is capable of sorting object keys in arbitrary orders, as well
as optional sorting the contents of arrays. You can configure this using a
JSON-based config file.

The config file should be a JSON object. It can contain three keys,
"indent", "keyOrder" and "arraySort". You can specify just one key as
well. Note that specifying "indent" in the config file will override any
command line.

The "keyOrder" key should in turn contain an object where the keys are JSON
Path expressions and the values are arrays of key names. The JSON Path
support is fairly limited.

All expressions must start with "$". You can use the following types of
expressions:

* \.\.  - This a recursive descent operator that matches any number of nodes of any type.
* .*  - This matches a single node of any type.
* [*] - This matches every element of an array.

When an object in the JSON file matches a path, it's keys are sorted as
specified. Note that if an object matches multiple JSON Path expressions the
results are unpredictable.

Here is an example config for JSON Schemas:

```json
{
    "indent": "    ",
    "keyOrder": {
        "$": [
            "$schema",
            "$id",
            "title",
            "description",
            "type",
            "additionalProperties",
            "properties",
            "required"
        ],
        "$..properties.*": [
            "$id",
            "title",
            "description",
            "type",
            "x-nullable",
            "enum",
            "format",
            "additionalProperties",
            "properties",
            "required",
            "examples"
        ]
    },
    "arraySort": [
        "$..properties..enum",
        "$..required"
    ]
}
```

The order of the key names in the arrays tell the tidier what order the keys
should be sorted in. Any keys not explicitly listed will be sorted _after_
the listed keys in case-insensitive alphanumeric order.

If you want to sort all of an object's keys in case-insensitive alphanumeric
order you can provide an empty array for the key order.

The "arraySort" key is an array of JSON Path expressions. Any array matching
the expression will be sorted numerically or as strings, as
appropriate. Strings are sorted in case-insensitive alphanumeric order.

* -check - Run in check mode. In this mode we exit 0 if all files are already tidy, otherwise the exit status is 1.
* -config - A config file containing key ordering and array sorting specifications.
* -debug - Enable debugging output.
* -ext - The file extension to match against. Only files with this extension will be tidied. (default ".json")
* -help - Show usage information.
* -indent - The string with which to indent JSON. Defaults to 4 spaces.
* -stdout - Instead of tidying file in place, output content to stdout. When this is flag is set the file contents will be printed even when it is already tidy. This flag is irrelevant when running in -check mode.
* -verbose - Be more verbose with output.
