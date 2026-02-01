package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}

const (
	LINE_PREFIX     = "│\t"
	TAB_PREFIX      = "\t"
	NOT_LAST_PREFIX = "├───"
	LAST_PREFIX     = "└───"
)

func dirTree(out io.Writer, path string, printFiles bool) error {
	return printTree(out, path, printFiles, "")
}

func printTree(out io.Writer, path string, printFiles bool, indentPrefix string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	filtered := entries
	if !printFiles {
		filtered = make([]os.DirEntry, 0, len(entries))
		for _, e := range entries {
			if e.IsDir() {
				filtered = append(filtered, e)
			}
		}
	}

	for i, entry := range filtered {
		isLast := i == len(filtered)-1

		var connector string
		if isLast {
			connector = LAST_PREFIX
		} else {
			connector = NOT_LAST_PREFIX
		}

		name := entry.Name()
		if !entry.IsDir() {
			info, err := entry.Info()
			if err != nil {
				return err
			}
			if info.Size() == 0 {
				name = name + " (empty)"
			} else {
				name = fmt.Sprintf("%s (%db)", name, info.Size())
			}
		}

		fmt.Fprintf(out, "%s%s%s\n", indentPrefix, connector, name)

		if entry.IsDir() {
			var nextIndentPrefix string
			if isLast {
				nextIndentPrefix = indentPrefix + TAB_PREFIX
			} else {
				nextIndentPrefix = indentPrefix + LINE_PREFIX
			}
			err := printTree(out, filepath.Join(path, entry.Name()), printFiles, nextIndentPrefix)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
