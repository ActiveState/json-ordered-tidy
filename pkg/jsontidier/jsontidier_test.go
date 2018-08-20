package jsontidier

import (
	"testing"

	"github.com/autarch/testify/assert"
)

func TestPreserveKeyOrde(t *testing.T) {
	orig := `
{"foo":42,"bar":"hello"}
`

	expect := `{
    "foo": 42,
    "bar": "hello"
}
`

	compareTidied(t, NewParams{}, orig, expect)

	orig = `
{"bar":"hello","foo":42}
`

	expect = `{
    "bar": "hello",
    "foo": 42
}
`

	compareTidied(t, NewParams{}, orig, expect)
}

func TestCustomIndent(t *testing.T) {
	orig := `
{"foo":42,"bar":"hello"}
`

	expect := `{
  "foo": 42,
  "bar": "hello"
}
`

	compareTidied(t, NewParams{Indent: stringRef("  ")}, orig, expect)
}

func TestTidyBytes(t *testing.T) {
	orig := []byte(`
{"foo":42,"bar":"hello"}
`)

	expect := []byte(`{
    "foo": 42,
    "bar": "hello"
}
`)

	jt := NewJSONTidier(NewParams{})
	tidied, err := jt.TidyBytes(orig)
	assert.Nil(t, err, "no error calling TidyBytes")
	assert.Equal(t, expect, tidied, "got expected tidied JSON")
}

func TestArraySorting(t *testing.T) {
	orig := `{
    "foo": {
        "bar": [ 3, 2, 4, 1 ]
    },
    "bar": [ 3, 2, 1, 4 ],
    "baz": [ "c", "d", "B", "a" ],
    "quux": [ 3, 2, 1, 4 ]
}`

	expect := `{
    "foo": {
        "bar": [
            1,
            2,
            3,
            4
        ]
    },
    "bar": [
        3,
        2,
        1,
        4
    ],
    "baz": [
        "a",
        "B",
        "c",
        "d"
    ],
    "quux": [
        1,
        2,
        3,
        4
    ]
}
`

	compareTidied(
		t,
		NewParams{ArraySort: []string{"$.foo.bar", "$.baz", "$..quux"}},
		orig,
		expect,
	)
}

func TestBasicKeySorting(t *testing.T) {
	orig := `{
    "foo": 1,
    "bar": 2,
    "baz": 3,
    "buz": 4,
    "quux": 5,
    "extra": 6,
    "and": 7
}`

	expect := `{
    "quux": 5,
    "buz": 4,
    "baz": 3,
    "bar": 2,
    "foo": 1,
    "and": 7,
    "extra": 6
}
`

	compareTidied(
		t,
		NewParams{KeyOrder: map[string][]string{
			"$": {"quux", "buz", "baz", "bar", "foo"},
		}},
		orig,
		expect,
	)
}

func TestJSONPathHandling(t *testing.T) {
	orig := `{
"k1": { "bar": { "baz": { "b": 1, "a": 2 } } },
"k2": { "bar": { "quux": { "baz": { "b": 1, "a": 2 } } } },
"k3": { "x": { "y": { "b": 1, "a": 2 } } },
"k4": { "x": { "z": { "y": { "b": 1, "a": 2 } } } },
"k5": [ { "b": 1, "a": 2, "c": 3 }, { "c": 1, "b": 2, "a": 3 } ]
}`

	expect := `{
    "k1": {
        "bar": {
            "baz": {
                "a": 2,
                "b": 1
            }
        }
    },
    "k2": {
        "bar": {
            "quux": {
                "baz": {
                    "a": 2,
                    "b": 1
                }
            }
        }
    },
    "k3": {
        "x": {
            "y": {
                "a": 2,
                "b": 1
            }
        }
    },
    "k4": {
        "x": {
            "z": {
                "y": {
                    "b": 1,
                    "a": 2
                }
            }
        }
    },
    "k5": [
        {
            "a": 2,
            "b": 1,
            "c": 3
        },
        {
            "a": 3,
            "b": 2,
            "c": 1
        }
    ]
}
`

	// This test is testing the difference between ".." and ".*". The ".."
	// path matches at any depth while ".*" matches only one level of
	// nesting. We also test the use of [*] as a selector.
	compareTidied(
		t,
		NewParams{KeyOrder: map[string][]string{
			"$..baz":  {"a", "b"},
			"$.*.*.y": {"a", "b"},
			"$.k5[*]": {"a", "b", "c"},
		}},
		orig,
		expect,
	)
}

func TestEmptyKeyOrderArray(t *testing.T) {
	orig := `{
"x": 42,
"z": 1,
"y": 2,
"B": 3,
"a": 4
}`

	expect := `{
    "a": 4,
    "B": 3,
    "x": 42,
    "y": 2,
    "z": 1
}
`

	compareTidied(
		t,
		NewParams{KeyOrder: map[string][]string{
			"$": {},
		}},
		orig,
		expect,
	)
}

func TestJSONSchema(t *testing.T) {
	orig := `{
    "required": [
        "os_name",
        "cpu_type",
        "thing_id"
    ],
    "properties": {
        "thing_id": {
            "format": "uuid",
            "type": "string"
        },
        "os_name": {
            "type": "string",
            "enum": [
                "linux",
                "AIX",
                "Solaris",
                "Windows",
                "HP-UX",
                "macOS"
            ]
        },
        "os_version": {
            "type": "string",
            "description": "The version of the operating system. This will be empty on Linux systems, where what we care about is the libc_version."
        },
        "processor": {
            "type": "object",
            "properties": {
                "cpu_type": {
                    "type": "string",
                    "enum": [
                        "x86",
                        "PowerPC",
                        "Sparc",
                        "IA64"
                    ]
                },
                "bit_width": {
                    "type": "string",
                    "enum": [
                        "64",
                        "32"
                    ]
                }
            },
            "additionalProperties": false,
            "required": [
                "cpu_type",
                "bit_width"
            ]
        },
        "libc_version": {
            "description": "This can be omitted except for linux systems. On non-linux systems this value is not meaningful.",
            "type": "string"
        },
        "end_of_support_date": {
            "x-nullable": true,
            "type": "string",
            "format": "date",
            "description": "If there is a planned end of support date for this thing, this will be populated."
        },
        "display_name": {
            "description": "A generalized description of the plaform suitable for display to end users.",
            "type": "string"
        }
    },
    "$schema": "http://json-schema.org/draft-04/schema#",
    "$id": "http://example.com/json-schemas/thing.json",
    "type": "object",
    "title": "Thing",
    "description": "It's a thing.",
    "additionalProperties": false
}`

	expect := `{
    "$schema": "http://json-schema.org/draft-04/schema#",
    "$id": "http://example.com/json-schemas/thing.json",
    "title": "Thing",
    "description": "It's a thing.",
    "type": "object",
    "additionalProperties": false,
    "properties": {
        "thing_id": {
            "type": "string",
            "format": "uuid"
        },
        "os_name": {
            "type": "string",
            "enum": [
                "AIX",
                "HP-UX",
                "linux",
                "macOS",
                "Solaris",
                "Windows"
            ]
        },
        "os_version": {
            "description": "The version of the operating system. This will be empty on Linux systems, where what we care about is the libc_version.",
            "type": "string"
        },
        "processor": {
            "type": "object",
            "additionalProperties": false,
            "properties": {
                "cpu_type": {
                    "type": "string",
                    "enum": [
                        "IA64",
                        "PowerPC",
                        "Sparc",
                        "x86"
                    ]
                },
                "bit_width": {
                    "type": "string",
                    "enum": [
                        "32",
                        "64"
                    ]
                }
            },
            "required": [
                "bit_width",
                "cpu_type"
            ]
        },
        "libc_version": {
            "description": "This can be omitted except for linux systems. On non-linux systems this value is not meaningful.",
            "type": "string"
        },
        "end_of_support_date": {
            "description": "If there is a planned end of support date for this thing, this will be populated.",
            "type": "string",
            "x-nullable": true,
            "format": "date"
        },
        "display_name": {
            "description": "A generalized description of the plaform suitable for display to end users.",
            "type": "string"
        }
    },
    "required": [
        "cpu_type",
        "os_name",
        "thing_id"
    ]
}
`

	compareTidied(
		t,
		NewParams{
			KeyOrder: map[string][]string{
				"$": {
					"$schema",
					"$id",
					"title",
					"description",
					"type",
					"additionalProperties",
					"properties",
					"required",
				},
				"$..properties.*": {
					"$id",
					"description",
					"type",
					"x-nullable",
					"enum",
					"format",
					"additionalProperties",
					"properties",
					"required",
					"examples",
				},
			},
			ArraySort: []string{
				"$..properties..enum",
				"$..required",
			},
		},
		orig,
		expect,
	)
}

func stringRef(s string) *string {
	return &s
}

func compareTidied(t *testing.T, np NewParams, orig, expect string) {
	jt := NewJSONTidier(np)
	tidied, err := jt.TidyString(orig)
	assert.Nil(t, err, "no error calling TidyString")
	assert.Equal(t, expect, tidied, "got expected tidied JSON")
}
