package ui

import (
	"strings"
)

// MarkLines provides a way to construct a styled text by separating the content
// and the styling.
//
// The arguments are groups of either
//
// * A single string, in which case it represents an unstyled line;
//
// * Three arguments that can be passed to MarkLine, in which case they are passed
//   to MarkLine and the return value is used as a styled line.
//
// Lines represented by all the groups are joined together.
//
// This function is mainly useful for constructing multi-line Text's with
// alignment across those lines. An example:
//
//   var stylesheet = map[rune]string{
//       '-': 'reverse',
//       'x': 'blue bg-green',
//   }
//   var text = FromMarkedLines(
//       "foo      bar      foobar", stylesheet,
//       "---      xxx      ------"
//       "lorem    ipsum    dolar",
//   )
func MarkLines(args ...interface{}) Text {
	var text Text
	for i := 0; i < len(args); i++ {
		line, ok := args[i].(string)
		if !ok {
			// TODO(xiaq): Surface the error.
			continue
		}
		if i > 0 {
			text = text.ConcatText(MakeText("\n"))
		}
		if i+2 < len(args) {
			if stylesheet, ok := args[i+1].(map[rune]string); ok {
				if style, ok := args[i+2].(string); ok {
					text = text.ConcatText(MarkText(line, stylesheet, style))
					i += 2
					continue
				}
			}
		}
		text = text.ConcatText(MakeText(line))
	}
	return text
}

// MarkText applies styles to all the runes in the line, using the runes in
// the style string. The stylesheet argument specifies which style each rune
// represents. The styles can be space-separated list of transformers.
func MarkText(line string, stylesheet map[rune]string, style string) Text {
	var text Text
	styleRuns := toRuns(style)
	for _, styleRun := range styleRuns {
		i := bytesForFirstNRunes(line, styleRun.n)
		text = text.ConcatText(
			MakeText(line[:i], strings.Split(stylesheet[styleRun.r], " ")...))
		line = line[i:]
	}
	if len(line) > 0 {
		text = text.ConcatText(MakeText(line))
	}
	return text
}

type run struct {
	r rune
	n int
}

func toRuns(s string) []run {
	var runs []run
	current := run{}
	for _, r := range s {
		if r != current.r {
			if current.n > 0 {
				runs = append(runs, current)
			}
			current = run{r, 1}
		} else {
			current.n++
		}
	}
	if current.n > 0 {
		runs = append(runs, current)
	}
	return runs
}

func bytesForFirstNRunes(s string, n int) int {
	k := 0
	for i, _ := range s {
		if k == n {
			return i
		}
		k++
	}
	return len(s)
}
