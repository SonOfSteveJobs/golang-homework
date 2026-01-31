package main

import (
	"io"
	"os"
	"path"
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

// решение
const (
	INDENT_PREFIX      = "│\t"
	CONNECTOR_LAST     = "└───"
	CONNECTOR_NOT_LAST = "├───"
)

type StackItem struct {
	pathname string
	prefix   string
	name     string
	isLast   bool
	isDir    bool
	size     int64
}

func dirTree(out io.Writer, pathname string, printFiles bool) error {
	stack := []StackItem{}
	stack = append(stack, StackItem{pathname: pathname, prefix: "", name: pathname, isLast: true, isDir: true})

	for len(stack) > 0 {
		item := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		out.Write([]byte(item.prefix + item.name + "\n"))

		if item.isDir {
			entry, err := os.ReadDir(item.pathname)
			if err != nil {
				return err
			}

			for i := len(entry) - 1; i >= 0; i-- {
				var size int64
				if !entry[i].IsDir() {
					info, err := entry[i].Info()
					if err != nil {
						return err
					}
					size = info.Size()
				}

				stack = append(stack, StackItem{
					pathname: path.Join(item.pathname, entry[i].Name()),
					prefix:   item.prefix + CONNECTOR_NOT_LAST,
					name:     entry[i].Name(),
					isLast:   i == 0,
					isDir:    entry[i].IsDir(),
					size:     size,
				})
			}
		}
	}

	return nil
}
