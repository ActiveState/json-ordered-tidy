package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	goj "gitlab.com/c0b/go-ordered-json"
)

type tidier struct {
	check     bool
	indent    string
	extRegexp *regexp.Regexp
	exit      int
}

func main() {
	var check bool
	flag.BoolVar(&check, "check", false, "Run in check mode. In this mode we exit 0 if all files are already tidy, otherwise the exit status is 1.")
	var indent string
	flag.StringVar(&indent, "indent", "    ", "The string with which to indent JSON. Defaults to 4 spaces.")
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

	t := tidier{
		check:     check,
		indent:    indent,
		extRegexp: regexp.MustCompile(regexp.QuoteMeta(ext) + `$`),
		exit:      0,
	}

	for _, path := range flag.Args() {
		t.handlePath(path)
	}

	os.Exit(t.exit)
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

`)
	flag.PrintDefaults()
}

func (t *tidier) handlePath(path string) {
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error stat'ing path %s: %s\n", path, err)
		return
	}
	if fi.IsDir() {
		t.recurseDir(path)
	} else if t.extRegexp.MatchString(path) {
		t.tidy(fi, path)
	}
}

func (t *tidier) recurseDir(dir string) {
	f, err := os.Open(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open directory %s: %s\n", dir, err)
		return
	}

	names, err := f.Readdirnames(0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read directory contents of %s: %s\n", dir, err)
		return
	}

	fmt.Fprintf(os.Stdout, "Looking in %s for JSON files\n", dir)

	for _, n := range names {
		t.handlePath(filepath.Join(dir, n))
	}
}

func (t *tidier) tidy(fi os.FileInfo, file string) {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not read file %s: %s\n", file, err)
		return
	}

	var om *goj.OrderedMap = goj.NewOrderedMap()
	err = json.Unmarshal(c, om)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse JSON from %s into an OrderedMap struct: %s\n", file, err)
		return
	}

	j, err := json.MarshalIndent(om, "", t.indent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not marshal JSON from OrderedMap struct: %s\n", err)
		return
	}

	j = append(j, '\n')

	if string(c) != string(j) {
		if t.check {
			fmt.Fprintf(os.Stdout, "%s is not tidy\n", file)
			t.exit = 1
		} else {
			err = ioutil.WriteFile(file, j, fi.Mode().Perm())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Could not write tidied JSON to %s: %s\n", file, err)
				return
			}

			fmt.Fprintf(os.Stdout, "Tidied %s\n", file)
		}
	} else {
		fmt.Fprintf(os.Stdout, "%s is already tidy\n", file)
	}
}
