package commentutil

import "io"

// GetCodeFenceLength returns the length of a code fence needed to wrap code.
// A test suggestion that uses four backticks w/o code fence block.
// Fixes: https://github.com/reviewdog/reviewdog/issues/999
//
// Code fenced blocks are supported by GitHub Flavor Markdown.
// A code fence is typically three backticks.
//
//     ```
//     code
//     ```
//
// However, we sometimes need more backticks.
// https://docs.github.com/en/github/writing-on-github/working-with-advanced-formatting/creating-and-highlighting-code-blocks#fenced-code-blocks
//
// > To display triple backticks in a fenced code block, wrap them inside quadruple backticks.
// >
// >     ````
// >     ```
// >     Look! You can see my backticks.
// >     ```
// >     ````
func GetCodeFenceLength(code string) int {
	backticks := countBackticks(code) + 1
	if backticks < 3 {
		// At least three backticks are required.
		// https://github.github.com/gfm/#fenced-code-blocks
		// > A code fence is a sequence of at least three consecutive backtick characters (`) or tildes (~). (Tildes and backticks cannot be mixed.)
		backticks = 3
	}
	return backticks
}

// WriteCodeFence writes a code fence to w.
func WriteCodeFence(w io.Writer, length int) error {
	if w, ok := w.(io.ByteWriter); ok {
		// use WriteByte instead of Write to avoid memory allocation.
		for i := 0; i < length; i++ {
			if err := w.WriteByte('`'); err != nil {
				return err
			}
		}
		return nil
	}

	buf := make([]byte, length)
	for i := range buf {
		buf[i] = '`'
	}
	_, err := w.Write(buf)
	return err
}

// find code fences in s, and returns the maximum length of them.
func countBackticks(s string) int {
	inBackticks := true

	var count int
	var maxCount int
	for _, r := range s {
		if inBackticks {
			if r == '`' {
				count++
			} else {
				inBackticks = false
				if count > maxCount {
					maxCount = count
				}
				count = 0
			}
		}
		if r == '\n' {
			inBackticks = true
			count = 0
		}
	}
	if count > maxCount {
		maxCount = count
	}
	return maxCount
}
