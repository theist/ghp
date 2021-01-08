package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"
)

func max(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

func ellipseStr(str string, size int) string {
	runes := bytes.Runes([]byte(str))
	if len(runes) > size {
		return string(runes[:size-3]) + "..."
	}
	return string(runes)
}

func fancyNoteStr(n *note, idField, maxSize int) string {
	str := ""
	idSp := strings.Repeat(" ", idField-utf8.RuneCountInString("note:"))
	noteText := strings.Split(n.text, "\n")[0]
	remain := maxSize - idField
	noteText = ellipseStr(noteText, remain)
	str = "  note:" + idSp + noteText
	return str
}

func fancyIssueStr(i *issue, idField, asigneeField, maxSize int) string {
	//log.Printf("fancy str for %v", i.ghIssue.GetTitle())
	str := ""
	idSp := strings.Repeat(" ", idField-utf8.RuneCountInString(fmt.Sprintf("%v#%v", i.repository.GetName(), i.ghIssue.GetNumber())))
	asigneeSp := strings.Repeat(" ", asigneeField-utf8.RuneCountInString(fmt.Sprintf("@%v", i.ghIssue.GetAssignee().GetLogin())))
	flexSp := ""
	ellipsedTitle := ""
	omitLabels := false
	remain := maxSize - idField - asigneeField
	if remain > (i.lenLabelString() + 1 + utf8.RuneCountInString(i.ghIssue.GetTitle())) {
		flexSp = strings.Repeat(" ", remain-(1+i.lenLabelString()+utf8.RuneCountInString(i.ghIssue.GetTitle())))
		ellipsedTitle = i.ghIssue.GetTitle()
	} else if remain < 12+i.lenLabelString() {
		omitLabels = true
		if remain > (utf8.RuneCountInString(i.ghIssue.GetTitle())) {
			flexSp = strings.Repeat(" ", remain-utf8.RuneCountInString(i.ghIssue.GetTitle()))
		}
		ellipsedTitle = ellipseStr(i.ghIssue.GetTitle(), remain-1)
	} else {
		ellipsedTitle = ellipseStr(i.ghIssue.GetTitle(), remain-i.lenLabelString()-1)
	}
	str = "  " + i.repository.GetName() + "#" + strconv.Itoa(i.ghIssue.GetNumber()) + idSp + ellipsedTitle + flexSp
	if !omitLabels {
		str = str + " " + i.labelString()
	}
	if i.ghIssue.GetAssignee().GetLogin() != "" {
		str = str + " " + asigneeSp + "@" + i.ghIssue.GetAssignee().GetLogin()
	}
	return str
}

func fancyList(p *ProjectProxy, filter [][]string) {
	maxID := 0
	maxAsignee := 0
	cmax := consoleWidth()
	for _, col := range p.columns {
		for _, card := range col.cards {
			if card.match(filter) {
				i, isIssue := card.(*issue)
				if isIssue {
					idLen := utf8.RuneCountInString(fmt.Sprintf("%v#%v", i.repository.GetName(), i.ghIssue.GetNumber()))
					maxID = max(maxID, idLen)
					asLen := utf8.RuneCountInString(i.ghIssue.GetAssignee().GetLogin())
					maxAsignee = max(maxAsignee, asLen)
				}
			}
		}
	}
	for _, col := range p.columns {
		fmt.Printf("\n%v:\n", col.name)
		for _, card := range col.cards {
			if card.match(filter) {
				switch v := card.(type) {
				case *issue:
					fmt.Println(fancyIssueStr(v, maxID+1, maxAsignee+2, cmax-3))
				case *note:
					fmt.Println(fancyNoteStr(v, maxID+1, cmax-3))
				default:
					fmt.Printf("no case match for %#v", v)
				}
			}
		}
	}
}
