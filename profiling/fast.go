// CPU:
//
//	go test -bench=BenchmarkFast -cpuprofile=cpu.out
//	go tool pprof -http=:8083 cpu.out
//
// Memory:
//
//	go test -bench=BenchmarkFast -memprofile=mem.out
//	go tool pprof -http=:8084 mem.out
package main

import (
	"bufio"
	"fmt"
	"hw3/user"
	"io"
	"os"
	"strings"
)

// вам надо написать более быструю оптимальную этой функции
func FastSearch(out io.Writer) {
	/*
		!!! !!! !!!
		обратите внимание - в задании обязательно нужен отчет
		делать его лучше в самом начале, когда вы видите уже узкие места, но еще не оптимизировалм их
		так же обратите внимание на команду в параметром -http
		перечитайте еще раз задание
		!!! !!! !!!

		ОТЧЕТ В report.md ЕСЛИ ЧЕ
	*/
	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	usersSb := strings.Builder{}
	seenBrowsersSet := make(map[string]struct{})

	currentLineIndex := 0
	for scanner.Scan() {
		line := scanner.Bytes()

		var currentUser user.User
		currentUser.UnmarshalJSON(line)

		isAndroid := false
		isMSIE := false

		for _, browser := range currentUser.Browsers {
			if strings.Contains(browser, "Android") {
				isAndroid = true
				if _, ok := seenBrowsersSet[browser]; !ok {
					seenBrowsersSet[browser] = struct{}{}
				}
			}
			if strings.Contains(browser, "MSIE") {
				isMSIE = true
				if _, ok := seenBrowsersSet[browser]; !ok {
					seenBrowsersSet[browser] = struct{}{}
				}
			}
		}

		if isAndroid && isMSIE {
			replacedEmail := strings.ReplaceAll(currentUser.Email, "@", " [at] ")
			fmt.Fprintf(&usersSb, "[%d] %s <%s>\n", currentLineIndex, currentUser.Name, replacedEmail)
		}

		currentLineIndex++
	}

	fmt.Fprintln(out, "found users:\n"+usersSb.String())
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsersSet))
}
