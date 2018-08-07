package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"time"
)

const maxLines = 1024

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getTempFile(tmpDir *string) *os.File {
	tmpfile, err := ioutil.TempFile(*tmpDir, "randomize")
	if err != nil {
		log.Fatal(err)
	}
	return tmpfile
}

func sortandwrite(tmpDir *string, lineBuffer *buffer) *os.File {

	rand.Shuffle(lineBuffer.length, func(i, j int) {
		lineBuffer.buffer[i], lineBuffer.buffer[j] =
			lineBuffer.buffer[j], lineBuffer.buffer[i]
	})

	tmpfile := getTempFile(tmpDir)

	writer := bufio.NewWriter(tmpfile)
	for i := 0; i < lineBuffer.length; i++ {
		writer.WriteString(lineBuffer.buffer[i])
		writer.WriteString("\n")
	}
	writer.Flush()
	tmpfile.Close()
	return tmpfile
}

type rahScanner struct {
	scanner *bufio.Scanner
	rnd     int
}

func (rah *rahScanner) read() {
	if !rah.scanner.Scan() {
		rah.scanner = nil
	} else {
		rah.rnd = rand.Int()
	}
}

func merge(dest *os.File, source []*os.File) {
	// Fill up our read ahead scanners
	scanners := make([]*rahScanner, len(source))
	for i, file := range source {
		fp, err := os.Open(file.Name())
		if err != nil {
			log.Fatal(err)
		}
		defer fp.Close()

		scanners[i] = &rahScanner{scanner: bufio.NewScanner(fp)}
		scanners[i].read()
	}

	writer := bufio.NewWriter(dest)
	for {
		var largestScanner *rahScanner
		for _, scanner := range scanners {
			if scanner.scanner == nil {
				continue
			}
			if largestScanner == nil || scanner.rnd > largestScanner.rnd {
				largestScanner = scanner
			}
		}
		if largestScanner == nil { // all done
			break
		}
		writer.WriteString(largestScanner.scanner.Text())
		writer.WriteString("\n")
		largestScanner.read()
	}
	writer.Flush()
}

type buffer struct {
	buffer []string
	length int
}

func fillBuffer(scanner *bufio.Scanner, lineBuffer *buffer) bool {
	for lineBuffer.length = 0; lineBuffer.length < maxLines && scanner.Scan(); lineBuffer.length++ {
		lineBuffer.buffer[lineBuffer.length] = scanner.Text()
	}
	return lineBuffer.length != 0
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	tmpDir := flag.String("temp", "", "temp directory")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: randomize [options]\n")
		fmt.Fprintf(os.Stderr, "\n")
		fmt.Fprintf(os.Stderr, "Randomize records (lines) of text\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)

	writtenFiles := make([]*os.File, 0, 32)
	defer func() {
		for _, fp := range writtenFiles {
			os.Remove(fp.Name())
		}
	}()

	lineBuffer := &buffer{buffer: make([]string, maxLines, maxLines), length: 0}

	// Start reading and randomize into small batches
	for fillBuffer(scanner, lineBuffer) {
		writtenFiles = append(writtenFiles, sortandwrite(tmpDir, lineBuffer))
	}

	// If we didn't read any data, we're done
	if len(writtenFiles) == 0 {
		return
	}

	// Randomize the batches
	rand.Shuffle(len(writtenFiles), func(i, j int) {
		writtenFiles[i], writtenFiles[j] = writtenFiles[j], writtenFiles[i]
	})

	// Chunk the batches and start merging them
	// we're done if we're left with one file
	for len(writtenFiles) != 1 {
		sliceSize := min(32, len(writtenFiles))

		// Append the dest merge tmp file to the end
		mergeTmpFile := getTempFile(tmpDir)
		writtenFiles = append(writtenFiles, mergeTmpFile)

		// Merge the slice
		chunk := writtenFiles[0:sliceSize]
		merge(mergeTmpFile, chunk)
		mergeTmpFile.Close()

		// Delete merged files
		for _, f := range chunk {
			os.Remove(f.Name())
		}

		// Remove the merged files from the head
		writtenFiles = writtenFiles[sliceSize:]
	}

	// Print the last randomized file and we're done
	fd, err := os.Open(writtenFiles[0].Name())
	if err != nil {
		log.Fatal(err)
	}
	io.Copy(os.Stdout, fd)
}
