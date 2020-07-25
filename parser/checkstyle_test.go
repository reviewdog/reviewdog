package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
)

func ExampleCheckStyleParser() {
	const sample = `<?xml version="1.0" encoding="utf-8"?><checkstyle version="4.3"><file name="/path/to/file"><error line="1" column="10" severity="error" message="&apos;addOne&apos; is defined but never used. (no-unused-vars)" source="eslint.rules.no-unused-vars" /><error line="2" column="9" severity="error" message="Use the isNaN function to compare with NaN. (use-isnan)" source="eslint.rules.use-isnan" /><error line="3" column="16" severity="error" message="Unexpected space before unary operator &apos;++&apos;. (space-unary-ops)" source="eslint.rules.space-unary-ops" /><error line="3" column="20" severity="warning" message="Missing semicolon. (semi)" source="eslint.rules.semi" /><error line="4" column="12" severity="warning" message="Unnecessary &apos;else&apos; after &apos;return&apos;. (no-else-return)" source="eslint.rules.no-else-return" /><error line="5" column="7" severity="warning" message="Expected indentation of 8 spaces but found 6. (indent)" source="eslint.rules.indent" /><error line="5" column="7" severity="error" message="Expected a return value. (consistent-return)" source="eslint.rules.consistent-return" /><error line="5" column="13" severity="warning" message="Missing semicolon. (semi)" source="eslint.rules.semi" /><error line="7" column="2" severity="error" message="Unnecessary semicolon. (no-extra-semi)" source="eslint.rules.no-extra-semi" /></file></checkstyle>`

	p := NewCheckStyleParser()
	diagnostics, err := p.Parse(strings.NewReader(sample))
	if err != nil {
		panic(err)
	}
	for _, d := range diagnostics {
		rdjson, _ := protojson.MarshalOptions{Indent: "  "}.Marshal(d)
		var out bytes.Buffer
		json.Indent(&out, rdjson, "", "  ")
		fmt.Println(out.String())
	}
	// Output:
	// {
	//   "message": "'addOne' is defined but never used. (no-unused-vars)",
	//   "location": {
	//     "path": "/path/to/file",
	//     "range": {
	//       "start": {
	//         "line": 1,
	//         "column": 10
	//       }
	//     }
	//   },
	//   "severity": "ERROR",
	//   "code": {
	//     "value": "eslint.rules.no-unused-vars"
	//   },
	//   "originalOutput": "/path/to/file:1:10: error: 'addOne' is defined but never used. (no-unused-vars) (eslint.rules.no-unused-vars)"
	// }
	// {
	//   "message": "Use the isNaN function to compare with NaN. (use-isnan)",
	//   "location": {
	//     "path": "/path/to/file",
	//     "range": {
	//       "start": {
	//         "line": 2,
	//         "column": 9
	//       }
	//     }
	//   },
	//   "severity": "ERROR",
	//   "code": {
	//     "value": "eslint.rules.use-isnan"
	//   },
	//   "originalOutput": "/path/to/file:2:9: error: Use the isNaN function to compare with NaN. (use-isnan) (eslint.rules.use-isnan)"
	// }
	// {
	//   "message": "Unexpected space before unary operator '++'. (space-unary-ops)",
	//   "location": {
	//     "path": "/path/to/file",
	//     "range": {
	//       "start": {
	//         "line": 3,
	//         "column": 16
	//       }
	//     }
	//   },
	//   "severity": "ERROR",
	//   "code": {
	//     "value": "eslint.rules.space-unary-ops"
	//   },
	//   "originalOutput": "/path/to/file:3:16: error: Unexpected space before unary operator '++'. (space-unary-ops) (eslint.rules.space-unary-ops)"
	// }
	// {
	//   "message": "Missing semicolon. (semi)",
	//   "location": {
	//     "path": "/path/to/file",
	//     "range": {
	//       "start": {
	//         "line": 3,
	//         "column": 20
	//       }
	//     }
	//   },
	//   "severity": "WARNING",
	//   "code": {
	//     "value": "eslint.rules.semi"
	//   },
	//   "originalOutput": "/path/to/file:3:20: warning: Missing semicolon. (semi) (eslint.rules.semi)"
	// }
	// {
	//   "message": "Unnecessary 'else' after 'return'. (no-else-return)",
	//   "location": {
	//     "path": "/path/to/file",
	//     "range": {
	//       "start": {
	//         "line": 4,
	//         "column": 12
	//       }
	//     }
	//   },
	//   "severity": "WARNING",
	//   "code": {
	//     "value": "eslint.rules.no-else-return"
	//   },
	//   "originalOutput": "/path/to/file:4:12: warning: Unnecessary 'else' after 'return'. (no-else-return) (eslint.rules.no-else-return)"
	// }
	// {
	//   "message": "Expected indentation of 8 spaces but found 6. (indent)",
	//   "location": {
	//     "path": "/path/to/file",
	//     "range": {
	//       "start": {
	//         "line": 5,
	//         "column": 7
	//       }
	//     }
	//   },
	//   "severity": "WARNING",
	//   "code": {
	//     "value": "eslint.rules.indent"
	//   },
	//   "originalOutput": "/path/to/file:5:7: warning: Expected indentation of 8 spaces but found 6. (indent) (eslint.rules.indent)"
	// }
	// {
	//   "message": "Expected a return value. (consistent-return)",
	//   "location": {
	//     "path": "/path/to/file",
	//     "range": {
	//       "start": {
	//         "line": 5,
	//         "column": 7
	//       }
	//     }
	//   },
	//   "severity": "ERROR",
	//   "code": {
	//     "value": "eslint.rules.consistent-return"
	//   },
	//   "originalOutput": "/path/to/file:5:7: error: Expected a return value. (consistent-return) (eslint.rules.consistent-return)"
	// }
	// {
	//   "message": "Missing semicolon. (semi)",
	//   "location": {
	//     "path": "/path/to/file",
	//     "range": {
	//       "start": {
	//         "line": 5,
	//         "column": 13
	//       }
	//     }
	//   },
	//   "severity": "WARNING",
	//   "code": {
	//     "value": "eslint.rules.semi"
	//   },
	//   "originalOutput": "/path/to/file:5:13: warning: Missing semicolon. (semi) (eslint.rules.semi)"
	// }
	// {
	//   "message": "Unnecessary semicolon. (no-extra-semi)",
	//   "location": {
	//     "path": "/path/to/file",
	//     "range": {
	//       "start": {
	//         "line": 7,
	//         "column": 2
	//       }
	//     }
	//   },
	//   "severity": "ERROR",
	//   "code": {
	//     "value": "eslint.rules.no-extra-semi"
	//   },
	//   "originalOutput": "/path/to/file:7:2: error: Unnecessary semicolon. (no-extra-semi) (eslint.rules.no-extra-semi)"
	// }
}
