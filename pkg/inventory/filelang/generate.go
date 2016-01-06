// +build ignore

package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/shurcooL/go-goon"
	"gopkg.in/yaml.v2"
	"src.sourcegraph.com/sourcegraph/pkg/inventory/filelang"
)

var (
	pkg = flag.String("pkg", "filelang", "package name of generated .go file")
	in  = flag.String("in", "_data", "input directory (with languages.yml, etc.)")
	out = flag.String("out", "data_%s.go", "output filename pattern")

	header = fmt.Sprintf("// Code generated by \"go run generate.go %s\"; DO NOT EDIT\n", strings.Join(os.Args[1:], " "))
)

func main() {
	flag.Parse()

	if err := generateLanguages(); err != nil {
		fmt.Fprintf(os.Stderr, "generating languages: error: %s\n", err)
		os.Exit(1)
	}

	if err := generateVendor(); err != nil {
		fmt.Fprintf(os.Stderr, "generating vendor: error: %s\n", err)
		os.Exit(1)
	}
}

func generateLanguages() error {
	input, err := ioutil.ReadFile(filepath.Join(*in, "languages.yml"))
	if err != nil {
		return err
	}
	var langs filelang.Languages
	if err := yaml.Unmarshal(input, &langs); err != nil {
		return err
	}
	src, err := generateCode("Langs", "Langs is a highly comprehensive list of programming languages.", langs)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(fmt.Sprintf(*out, "languages")), src, 0600)
}

func generateVendor() error {
	input, err := ioutil.ReadFile(filepath.Join(*in, "vendor.yml"))
	if err != nil {
		return err
	}
	var vendorPatterns []string
	if err := yaml.Unmarshal(input, &vendorPatterns); err != nil {
		return err
	}

	var buf bytes.Buffer
	printHeader(&buf)
	fmt.Fprintln(&buf, `import "regexp"`)
	fmt.Fprintln(&buf)
	fmt.Fprintln(&buf, "var vendorPatterns = []*regexp.Regexp{")
	for _, p := range vendorPatterns {
		fmt.Fprintf(&buf, "\tregexp.MustCompile(%q),\n", p)
	}
	fmt.Fprintln(&buf, "}")

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(fmt.Sprintf(*out, "vendor")), src, 0600)
}

func generateCode(varName, doc string, data interface{}) ([]byte, error) {
	var buf bytes.Buffer
	printHeader(&buf)
	if doc != "" {
		fmt.Fprintf(&buf, "// %s\n", doc)
	}
	fmt.Fprintf(&buf, "var %s = ", varName)
	if _, err := goon.Fdump(&buf, data); err != nil {
		return nil, err
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, err
	}

	if *pkg == "filelang" {
		src = bytes.Replace(src, []byte("filelang."), nil, -1)
	}

	return src, nil
}

func printHeader(w io.Writer) {
	fmt.Fprintln(w, header)
	fmt.Fprintf(w, "package %s\n\n", *pkg)
}
