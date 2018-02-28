package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/xeipuuv/gojsonschema"
)

func main() {
	var dir string
	flag.StringVar(&dir, "dir", "", "The directory containing the schemas to validate.")
	var verbose bool
	flag.BoolVar(&verbose, "verbose", false, "Print verbose output.")
	var help bool
	flag.BoolVar(&help, "help", false, "Show usage information.")
	flag.Parse()

	if help {
		usage("")
		os.Exit(0)
	}

	if dir == "" {
		usage("you must specify a -dir to validate")
		os.Exit(1)
	}

	if !dirExists(dir) {
		exitError("there is no directory at %s", dir)
	}

	exit := 0
	files := allSchemas(dir)
	if len(files) == 0 {
		exitError("did not find any JSON schemas in %s", dir)
	}
	for _, f := range files {
		if verbose {
			fmt.Printf("Checking %s\n", f)
		}
		schema := readOrFatal(f)
		l := gojsonschema.NewStringLoader(schema)
		_, err := gojsonschema.NewSchema(l)
		if err != nil {
			exit++
			fmt.Fprintf(os.Stderr, "%s does not contain a valid JSON schema:\n  %v\n\n", f, err)
		}
		if verbose {
			fmt.Println("  ok")
		}
	}

	os.Exit(exit)
}

func usage(err string) {
	if err != "" {
		fmt.Printf("\n  *** %s ***\n", err)
	}

	fmt.Print(
		`
  check-json-schemas -dir ./schemas

  This command validates that all the JSON files in a directory contain a
  schema that github.com/xeipuuv/gojsonschema can load and validate documents
  against.

`)
	flag.PrintDefaults()
}

func dirExists(file string) bool {
	_, err := os.Stat(file)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}

	exitError("error calling stat on %s", err)

	// We'll never get here
	return false
}

func allSchemas(dir string) []string {
	var files []string
	err := filepath.Walk(dir, func(path string, _ os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == ".git" || path == "vendor" {
			return filepath.SkipDir
		}

		if regexp.MustCompile(`\.json$`).MatchString(path) {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		log.Fatalf("Error walking tree: %v", err)
	}

	return files
}

func exitError(out string, args ...interface{}) {
	fmt.Printf("  validate-json-schema: "+out+"\n", args...)
	os.Exit(2)
}

func readOrFatal(file string) string {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		exitError("Could not read %s: %v", file, err)
	}
	return string(b)
}
