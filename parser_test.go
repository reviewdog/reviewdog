package reviewdog

import (
	"strings"
	"testing"
)

func TestNewErrorformatParserString(t *testing.T) {
	in := []string{`%f:%l:%c:%m`, `%-G%.%#`}

	got, err := NewErrorformatParserString(in)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.efm.Efms) != len(in) {
		t.Errorf("NewErrorformatParserString: len: got %v, want %v", len(got.efm.Efms), len(in))
	}
}

func TestCheckStyleParser(t *testing.T) {
	const sample = `<?xml version="1.0" encoding="utf-8"?><checkstyle version="4.3"><file name="/path/to/file"><error line="1" column="10" severity="error" message="&apos;addOne&apos; is defined but never used. (no-unused-vars)" source="eslint.rules.no-unused-vars" /><error line="2" column="9" severity="error" message="Use the isNaN function to compare with NaN. (use-isnan)" source="eslint.rules.use-isnan" /><error line="3" column="16" severity="error" message="Unexpected space before unary operator &apos;++&apos;. (space-unary-ops)" source="eslint.rules.space-unary-ops" /><error line="3" column="20" severity="warning" message="Missing semicolon. (semi)" source="eslint.rules.semi" /><error line="4" column="12" severity="warning" message="Unnecessary &apos;else&apos; after &apos;return&apos;. (no-else-return)" source="eslint.rules.no-else-return" /><error line="5" column="7" severity="warning" message="Expected indentation of 8 spaces but found 6. (indent)" source="eslint.rules.indent" /><error line="5" column="7" severity="error" message="Expected a return value. (consistent-return)" source="eslint.rules.consistent-return" /><error line="5" column="13" severity="warning" message="Missing semicolon. (semi)" source="eslint.rules.semi" /><error line="7" column="2" severity="error" message="Unnecessary semicolon. (no-extra-semi)" source="eslint.rules.no-extra-semi" /></file></checkstyle>`

	wants := []string{

		"/path/to/file:1:10: error: 'addOne' is defined but never used. (no-unused-vars) (eslint.rules.no-unused-vars)",
		"/path/to/file:2:9: error: Use the isNaN function to compare with NaN. (use-isnan) (eslint.rules.use-isnan)",
		"/path/to/file:3:16: error: Unexpected space before unary operator '++'. (space-unary-ops) (eslint.rules.space-unary-ops)",
		"/path/to/file:3:20: warning: Missing semicolon. (semi) (eslint.rules.semi)",
		"/path/to/file:4:12: warning: Unnecessary 'else' after 'return'. (no-else-return) (eslint.rules.no-else-return)",
		"/path/to/file:5:7: warning: Expected indentation of 8 spaces but found 6. (indent) (eslint.rules.indent)",
		"/path/to/file:5:7: error: Expected a return value. (consistent-return) (eslint.rules.consistent-return)",
		"/path/to/file:5:13: warning: Missing semicolon. (semi) (eslint.rules.semi)",
		"/path/to/file:7:2: error: Unnecessary semicolon. (no-extra-semi) (eslint.rules.no-extra-semi)",
	}

	p := NewCheckStyleParser()
	crs, err := p.Parse(strings.NewReader(sample))
	if err != nil {
		t.Error(err)
	}
	for i, cr := range crs {
		if got, want := cr.Lines[0], wants[i]; got != want {
			t.Errorf("%d: got %v, want %v", i, got, want)
		}
	}

}
