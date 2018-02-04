package eval

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/elves/elvish/eval/types"
	"github.com/elves/elvish/util"
)

// String operations.

var ErrInput = errors.New("input error")

func init() {
	addToReflectBuiltinFns(map[string]interface{}{
		"<s":  func(a, b string) bool { return a < b },
		"<=s": func(a, b string) bool { return a <= b },
		"==s": func(a, b string) bool { return a == b },
		"!=s": func(a, b string) bool { return a != b },
		">s":  func(a, b string) bool { return a > b },
		">=s": func(a, b string) bool { return a >= b },

		"to-string": toString,

		"ord":  ord,
		"base": base,

		"wcswidth":          wcswidth,
		"-override-wcwidth": overrideWcwidth,

		"has-prefix": strings.HasPrefix,
		"has-suffix": strings.HasSuffix,
	})
	addToBuiltinFns([]*BuiltinFn{

		{"joins", joins},
		{"splits", splits},
		{"replaces", replaces},

		{"eawk", eawk},
	})
}

// toString converts all arguments to strings.
func toString(fm *Frame, args ...interface{}) {
	out := fm.OutputChan()
	for _, a := range args {
		out <- types.ToString(a)
	}
}

// joins joins all input strings with a delimiter.
func joins(ec *Frame, args []interface{}, opts map[string]interface{}) {
	var sepv string
	iterate := ScanArgsOptionalInput(ec, args, &sepv)
	sep := sepv
	TakeNoOpt(opts)

	var buf bytes.Buffer
	iterate(func(v interface{}) {
		if s, ok := v.(string); ok {
			if buf.Len() > 0 {
				buf.WriteString(sep)
			}
			buf.WriteString(s)
		} else {
			throwf("join wants string input, got %s", types.Kind(v))
		}
	})
	out := ec.ports[1].Chan
	out <- buf.String()
}

// splits splits an argument strings by a delimiter and writes all pieces.
func splits(ec *Frame, args []interface{}, opts map[string]interface{}) {
	var (
		s, sep string
		optMax int
	)
	ScanArgs(args, &sep, &s)
	ScanOpts(opts, OptToScan{"max", &optMax, "-1"})

	out := ec.ports[1].Chan
	parts := strings.SplitN(s, sep, optMax)
	for _, p := range parts {
		out <- p
	}
}

func replaces(ec *Frame, args []interface{}, opts map[string]interface{}) {
	var (
		old, repl, s string
		optMax       int
	)
	ScanArgs(args, &old, &repl, &s)
	ScanOpts(opts, OptToScan{"max", &optMax, "-1"})

	ec.ports[1].Chan <- strings.Replace(s, old, repl, optMax)
}

func ord(fm *Frame, s string) {
	out := fm.ports[1].Chan
	for _, r := range s {
		out <- "0x" + strconv.FormatInt(int64(r), 16)
	}
}

// ErrBadBase is thrown by the "base" builtin if the base is smaller than 2 or
// greater than 36.
var ErrBadBase = errors.New("bad base")

func base(fm *Frame, b int, nums ...int) error {
	if b < 2 || b > 36 {
		return ErrBadBase
	}

	out := fm.ports[1].Chan
	for _, num := range nums {
		out <- strconv.FormatInt(int64(num), b)
	}
	return nil
}

func wcswidth(s string) string {
	return strconv.Itoa(util.Wcswidth(s))
}

func overrideWcwidth(s string, w int) error {
	r, err := toRune(s)
	if err != nil {
		return err
	}
	util.OverrideWcwidth(r, w)
	return nil
}

var eawkWordSep = regexp.MustCompile("[ \t]+")

// eawk takes a function. For each line in the input stream, it calls the
// function with the line and the words in the line. The words are found by
// stripping the line and splitting the line by whitespaces. The function may
// call break and continue. Overall this provides a similar functionality to
// awk, hence the name.
func eawk(ec *Frame, args []interface{}, opts map[string]interface{}) {
	var f Callable
	iterate := ScanArgsOptionalInput(ec, args, &f)
	TakeNoOpt(opts)

	broken := false
	iterate(func(v interface{}) {
		if broken {
			return
		}
		line, ok := v.(string)
		if !ok {
			throw(ErrInput)
		}
		args := []interface{}{line}
		for _, field := range eawkWordSep.Split(strings.Trim(line, " \t"), -1) {
			args = append(args, field)
		}

		newec := ec.fork("fn of eawk")
		// TODO: Close port 0 of newec.
		ex := newec.PCall(f, args, NoOpts)
		ClosePorts(newec.ports)

		if ex != nil {
			switch ex.(*Exception).Cause {
			case nil, Continue:
				// nop
			case Break:
				broken = true
			default:
				throw(ex)
			}
		}
	})
}
