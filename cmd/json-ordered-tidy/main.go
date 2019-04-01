package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ActiveState/json-tools/pkg/jsontidier"
)

type config struct {
	Indent    *string
	KeyOrder  map[string][]string
	ArraySort []string
}

type indentFlag struct {
	value string
	set   bool
}

func (f *indentFlag) Set(v string) error {
	f.value = v
	f.set = true
	return nil
}

func (f *indentFlag) String() string {
	return f.value
}

type program struct {
	stdout    bool
	check     bool
	verbose   bool
	debug     bool
	indent    indentFlag
	config    config
	extRegexp *regexp.Regexp
	exit      int
}

func main() {
	var indent indentFlag
	flag.Var(&indent, "indent", "The string with which to indent JSON. Defaults to 4 spaces.")
	var config string
	flag.StringVar(&config, "config", "", "A config file containing key ordering and array sorting specifications.")

	var stdout bool
	flag.BoolVar(&stdout, "stdout", false, "Instead of tidying file in place, output content to stdout. When this is flag is set the file contents will be printed even when it is already tidy. This flag is irrelevant when running in -check mode.")
	var check bool
	flag.BoolVar(&check, "check", false, "Run in check mode. In this mode we exit 0 if all files are already tidy, otherwise the exit status is 1.")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Be more verbose with output.")
	var debug bool
	flag.BoolVar(&debug, "debug", false, "Enable debugging output.")
	var ext string
	flag.StringVar(&ext, "ext", ".json", "The file extension to match against. Only files with this extension will be tidied.")
	var help bool
	flag.BoolVar(&help, "help", false, "Show usage information.")
	flag.Parse()

	if help {
		usage("")
		os.Exit(0)
	}

	a := flag.Args()
	if len(a) == 0 {
		usage("You did not provide any files or directories to tidy")
		os.Exit(1)
	}

	if stdout && check {
		usage("You passed both -stdout and -check, which do not make any sense together")
		os.Exit(1)
	}

	p := program{
		stdout:    stdout,
		check:     check,
		verbose:   verbose,
		indent:    indent,
		debug:     debug,
		extRegexp: regexp.MustCompile(regexp.QuoteMeta(ext) + `$`),
		exit:      0,
	}

	if config != "" {
		c, err := readConfigFile(config)
		if err != nil {
			usage(fmt.Sprintf("Error reading the config file you provided (%s): %s", config, err))
			os.Exit(1)
		}
		p.config = c

		if c.Indent != nil && indent.set && *c.Indent != indent.value {
			fmt.Fprintf(
				os.Stderr,
				"\n"+
					`  ** You set the --indent flag on the command line to "%s"`+
					"\n"+
					`  ** but you also set indent in your config file to "%s".`+
					"\n"+
					"  ** Using the value from your config file.\n\n",
				indent.value, *c.Indent,
			)
			p.exit = 1
		}
	}

	for _, path := range flag.Args() {
		p.handlePath(path)
	}

	os.Exit(p.exit)
}

func readConfigFile(path string) (config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return config{}, err
	}

	var c config
	err = json.Unmarshal(b, &c)
	if err != nil {
		return config{}, err
	}

	return c, nil
}

func usage(err string) {
	if err != "" {
		fmt.Printf("\n  *** %s ***\n", err)
	}

	fmt.Print(
		`
  json-ordered-tidy ... ./file.json ./dir

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

  ..  - This a recursive descent operator that matches any number of nodes
       of any type.

  .*  - This matches a single node of any type.

  [*] - This matches every element of an array.

  When an object in the JSON file matches a path, it's keys are sorted as
  specified. Note that if an object matches multiple JSON Path expressions the
  results are unpredictable.

  Here is an example config for JSON Schemas:

  {
	 "keyOrder":{
		"$":[
		   "$schema",
		   "$id",
		   "title",
		   "description",
		   "type",
		   "additionalProperties",
		   "properties",
		   "required"
		],
		"$..properties.*":[
		   "$id",
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
	 }
  }

  The order of the key names in the arrays tell the tidier what order the keys
  should be sorted in. Any keys not explicitly listed will be sorted _after_
  the listed keys in case-insensitive alphanumeric order.

  If you want to sort all of an object's keys in case-insensitive alphanumeric
  order you can provide an empty array for the key order.

  The "arraySort" key is an array of JSON Path expressions. Any array matching
  the expression will be sorted numerically or as strings, as
  appropriate. Strings are sorted in in case-insensitive alphanumeric order.

`)
	flag.PrintDefaults()
}

func (p *program) handlePath(path string) {
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error stat'ing path %s: %s\n", path, err)
		p.exit = 1
		return
	}
	if fi.IsDir() {
		p.recurseDir(path)
	} else if p.extRegexp.MatchString(path) {
		p.tidy(fi, path)
	}
}

func (p *program) recurseDir(dir string) {
	f, err := os.Open(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open directory %s: %s\n", dir, err)
		p.exit = 1
		return
	}

	names, err := f.Readdirnames(0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read directory contents of %s: %s\n", dir, err)
		p.exit = 1
		return
	}

	if p.verbose {
		fmt.Fprintf(os.Stdout, "Looking in %s for JSON files\n", dir)
	}

	for _, n := range names {
		p.handlePath(filepath.Join(dir, n))
	}
}

func (p *program) tidy(fi os.FileInfo, file string) {
	orig, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read file %s: %s\n", file, err)
		p.exit = 1
		return
	}

	np := jsontidier.NewParams{
		KeyOrder:  p.config.KeyOrder,
		ArraySort: p.config.ArraySort,
		Debug:     p.debug,
	}
	if p.config.Indent != nil {
		np.Indent = p.config.Indent
	} else if p.indent.set {
		np.Indent = &p.indent.value
	}
	jt := jsontidier.NewJSONTidier(np)

	tidied, err := jt.TidyBytes(orig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not tidy %s: %s\n", file, err)
		p.exit = 1
		return
	}

	if p.stdout {
		fmt.Print(string(tidied))
	}

	if string(orig) != string(tidied) {
		if p.check {
			fmt.Fprintf(os.Stdout, "%s is not tidy\n", file)
			p.exit = 1
		} else {
			err = ioutil.WriteFile(file, tidied, fi.Mode().Perm())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not write tidied JSON to %s: %s\n", file, err)
				p.exit = 1
				return
			}

			if p.verbose {
				fmt.Fprintf(os.Stdout, "Tidied %s\n", file)
			}
		}
	} else if p.verbose {
		fmt.Fprintf(os.Stdout, "%s is already tidy\n", file)
	}
}
