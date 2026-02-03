package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type User struct {
	Name          string   `json:"name"`
	Email         string   `json:"email"`
	Browsers      []string `json:"browsers"`
	Index         int
	IsAndroidUser bool
	IsMSIEUser    bool
}

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
		line := scanner.Text()

		var currentUser User
		err := json.Unmarshal([]byte(line), &currentUser)
		if err != nil {
			panic(err)
		}

		for _, browser := range currentUser.Browsers {
			if strings.Contains(browser, "Android") {
				currentUser.IsAndroidUser = true
				if _, ok := seenBrowsersSet[browser]; !ok {
					seenBrowsersSet[browser] = struct{}{}
				}
			}
			if strings.Contains(browser, "MSIE") {
				currentUser.IsMSIEUser = true
				if _, ok := seenBrowsersSet[browser]; !ok {
					seenBrowsersSet[browser] = struct{}{}
				}
			}
		}

		if currentUser.IsAndroidUser && currentUser.IsMSIEUser {
			replacedEmail := strings.ReplaceAll(currentUser.Email, "@", " [at] ")
			fmt.Fprintf(&usersSb, "[%d] %s <%s>\n", currentLineIndex, currentUser.Name, replacedEmail)
		}

		currentLineIndex++
	}

	fmt.Fprintln(out, "found users:\n"+usersSb.String())
	fmt.Fprintln(out, "Total unique browsers", len(seenBrowsersSet))
}
