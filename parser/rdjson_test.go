package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
)

func ExampleRDJSONParser() {
	const sample = `
{
  "source": {
    "name": "linter-name",
    "url": "https://github.com/reviewdog#linter-name"
  },
  "severity": "INFO",
  "diagnostics": [
    {
      "source": {
        "name": "deadcode"
      },
      "message": "'unused' is unused",
      "location": {
        "path": "testdata/main.go",
        "range": {
          "start": {
            "line": 18,
            "column": 6
          }
        }
      }
    },
    {
      "message": "printf: Sprintf format %d reads arg #1, but call has 0 args",
      "location": {
        "path": "testdata/main.go",
        "range": {
          "start": {
            "line": 13,
            "column": 2
          }
        }
      }
    },
    {
      "source": {
        "name": "severity-test"
      },
      "message": "severity test (string)",
      "location": {
        "path": "testdata/main.go",
        "range": {
          "start": {
            "line": 24,
            "column": 6
          }
        }
      },
      "severity": "WARNING"
    },
    {
      "source": {
        "name": "severity-test"
      },
      "message": "severity test (number)",
      "location": {
        "path": "testdata/main.go",
        "range": {
          "start": {
            "line": 24,
            "column": 6
          }
        }
      },
      "severity": 1
    }
  ]
}`
	p := NewRDJSONParser()
	diagnostics, err := p.Parse(strings.NewReader(sample))
	if err != nil {
		panic(err)
	}
	for _, d := range diagnostics {
		d.OriginalOutput = "" // Skip for testing as it's not deterministic.
		rdjson, _ := protojson.MarshalOptions{Indent: "  "}.Marshal(d)
		var out bytes.Buffer
		json.Indent(&out, rdjson, "", "  ")
		fmt.Println(out.String())
	}
	// Output:
	// {
	//   "message": "'unused' is unused",
	//   "location": {
	//     "path": "testdata/main.go",
	//     "range": {
	//       "start": {
	//         "line": 18,
	//         "column": 6
	//       }
	//     }
	//   },
	//   "severity": "INFO",
	//   "source": {
	//     "name": "deadcode"
	//   }
	// }
	// {
	//   "message": "printf: Sprintf format %d reads arg #1, but call has 0 args",
	//   "location": {
	//     "path": "testdata/main.go",
	//     "range": {
	//       "start": {
	//         "line": 13,
	//         "column": 2
	//       }
	//     }
	//   },
	//   "severity": "INFO",
	//   "source": {
	//     "name": "linter-name",
	//     "url": "https://github.com/reviewdog#linter-name"
	//   }
	// }
	// {
	//   "message": "severity test (string)",
	//   "location": {
	//     "path": "testdata/main.go",
	//     "range": {
	//       "start": {
	//         "line": 24,
	//         "column": 6
	//       }
	//     }
	//   },
	//   "severity": "WARNING",
	//   "source": {
	//     "name": "severity-test"
	//   }
	// }
	// {
	//   "message": "severity test (number)",
	//   "location": {
	//     "path": "testdata/main.go",
	//     "range": {
	//       "start": {
	//         "line": 24,
	//         "column": 6
	//       }
	//     }
	//   },
	//   "severity": "ERROR",
	//   "source": {
	//     "name": "severity-test"
	//   }
	// }
}
