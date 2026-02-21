package cli

import (
	"fmt"
	"io"
)

func printOK(out io.Writer, format string, args ...any) {
	printTagged(out, "[OK]", format, args...)
}

func printWarn(out io.Writer, format string, args ...any) {
	printTagged(out, "[WARN]", format, args...)
}

func printError(out io.Writer, format string, args ...any) {
	printTagged(out, "[ERROR]", format, args...)
}

func printHint(out io.Writer, format string, args ...any) {
	printTagged(out, "Hint:", format, args...)
}

func printTagged(out io.Writer, tag, format string, args ...any) {
	fmt.Fprintf(out, "%s %s\n", tag, fmt.Sprintf(format, args...))
}
