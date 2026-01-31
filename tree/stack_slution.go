// package main

// import (
// 	"fmt"
// 	"io"
// 	"os"
// 	"path/filepath"
// 	"strings"
// )

// func main() {
// 	out := os.Stdout
// 	if !(len(os.Args) == 2 || len(os.Args) == 3) {
// 		panic("usage go run main.go . [-f]")
// 	}
// 	path := os.Args[1]
// 	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
// 	err := dirTree(out, path, printFiles)
// 	if err != nil {
// 		panic(err.Error())
// 	}
// }

// const (
// 	INDENT_CONTINUE    = "│\t"
// 	INDENT_EMPTY       = "\t"
// 	CONNECTOR_LAST     = "└───"
// 	CONNECTOR_NOT_LAST = "├───"
// )

// type StackItem struct {
// 	pathname   string
// 	name       string
// 	isLast     bool
// 	isDir      bool
// 	size       int64
// 	indentMask []bool
// }

// func dirTree(out io.Writer, pathname string, printFiles bool) error {
// 	stack, err := getChildren(pathname, printFiles, []bool{})
// 	if err != nil {
// 		return err
// 	}

// 	for len(stack) > 0 {
// 		item := stack[len(stack)-1]
// 		stack = stack[:len(stack)-1]
// 		indent := buildIndent(item.indentMask)
// 		connector := CONNECTOR_NOT_LAST

// 		if item.isLast {
// 			connector = CONNECTOR_LAST
// 		}
// 		name := formatName(item.name, item.isDir, item.size)

// 		_, err := out.Write([]byte(indent + connector + name + "\n"))
// 		if err != nil {
// 			return err
// 		}

// 		if item.isDir {
// 			newMask := append(append([]bool{}, item.indentMask...), item.isLast)
// 			children, err := getChildren(item.pathname, printFiles, newMask)
// 			if err != nil {
// 				return err
// 			}
// 			stack = append(stack, children...)
// 		}
// 	}

// 	return nil
// }

// func getChildren(pathname string, printFiles bool, indentMask []bool) ([]StackItem, error) {
// 	entries, err := os.ReadDir(pathname)
// 	if err != nil {
// 		return nil, err
// 	}

// 	filtered := make([]os.DirEntry, 0, len(entries))
// 	for _, e := range entries {
// 		if e.IsDir() || printFiles {
// 			filtered = append(filtered, e)
// 		}
// 	}

// 	result := make([]StackItem, 0, len(filtered))
// 	for i := len(filtered) - 1; i >= 0; i-- {
// 		e := filtered[i]
// 		var size int64
// 		if !e.IsDir() {
// 			info, err := e.Info()
// 			if err != nil {
// 				return nil, err
// 			}
// 			size = info.Size()
// 		}
// 		result = append(result, StackItem{
// 			pathname:   filepath.Join(pathname, e.Name()),
// 			name:       e.Name(),
// 			isLast:     i == len(filtered)-1,
// 			isDir:      e.IsDir(),
// 			size:       size,
// 			indentMask: indentMask,
// 		})
// 	}
// 	return result, nil
// }

// func buildIndent(mask []bool) string {
// 	var sb strings.Builder
// 	for _, parentWasLast := range mask {
// 		if parentWasLast {
// 			sb.WriteString(INDENT_EMPTY)
// 		} else {
// 			sb.WriteString(INDENT_CONTINUE)
// 		}
// 	}
// 	return sb.String()
// }

// func formatName(name string, isDir bool, size int64) string {
// 	if isDir {
// 		return name
// 	}
// 	if size == 0 {
// 		return name + " (empty)"
// 	}
// 	return fmt.Sprintf("%s (%db)", name, size)
// }
