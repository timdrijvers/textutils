package fields

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type formatter interface {
	transform([]string) string
}
type literalFormatter struct {
	formatter
	literal string
}

type fieldFormatter struct {
	formatter
	index int
}

type reverseFieldFormatter struct {
	formatter
	index int
}

func (f *literalFormatter) transform(parts []string) string {
	return f.literal
}

func (f *fieldFormatter) transform(parts []string) string {
	if f.index < len(parts) {
		return parts[f.index]
	}
	return ""
}

func (f *reverseFieldFormatter) transform(parts []string) string {
	index := len(parts) + f.index
	if index >= 0 {
		return parts[index]
	}
	return ""
}

func getSplitter(separator *string) func(string) []string {

	if separator == nil || *separator == "" {
		return strings.Fields
	}

	return func(s string) []string {
		return strings.Split(s, *separator)
	}
}

func parsePattern(pattern string) []formatter {
	re, err := regexp.Compile("(\\$-?[0-9]+)")
	if err != nil {
		log.Fatal(err)
	}
	result := re.FindAllStringIndex(pattern, -1)

	previous := 0
	patternFormatter := make([]formatter, 0, 2*len(result))
	for _, match := range result {
		if match[0] > previous {
			patternFormatter = append(patternFormatter, &literalFormatter{literal: pattern[previous:match[0]]})
		}
		index, _ := strconv.Atoi(pattern[match[0]+1 : match[1]])
		if index < 0 {
			patternFormatter = append(patternFormatter, &reverseFieldFormatter{index: index})
		} else {
			patternFormatter = append(patternFormatter, &fieldFormatter{index: index})
		}
		previous = match[1]
	}

	if previous < len(pattern) {
		patternFormatter = append(patternFormatter, &literalFormatter{literal: pattern[previous:]})
	}
	return patternFormatter
}

func execute(formatters []formatter, fields []string) {
	for _, formatter := range formatters {
		fmt.Print(formatter.transform(fields))
	}
	fmt.Print("\n")
}

func main() {
	separator := flag.String("separator", "", "field separators")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: fields [options] <format>\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Format consist of a string that serves as template for each\n")
		fmt.Fprintf(os.Stderr, "line of stdin. Fields index is zero based.\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Format escape codes:\n")
		fmt.Fprintf(os.Stderr, "  $0 ... $n : field 0 .. n\n")
		fmt.Fprintf(os.Stderr, "  $-1 ... $-n : field n .. 0 (i.e. backwards)\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	leftArgs := flag.Args()
	if len(leftArgs) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	formatters := parsePattern(leftArgs[0])

	scanner := bufio.NewScanner(os.Stdin)
	splitter := getSplitter(separator)
	for scanner.Scan() {
		fields := splitter(scanner.Text())
		execute(formatters, fields)
	}
}
