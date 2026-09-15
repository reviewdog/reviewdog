package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"os"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestExampleSarifParser(t *testing.T) {
	p := NewSarifParser()
	for i, fixture := range fixtures {
		diagnostics, err := p.Parse(strings.NewReader(fixture[0]))
		if err != nil {
			panic(err)
		}
		if len(diagnostics) == 0 {
			t.Errorf("empty diagnostics")
		}
		for _, d := range diagnostics {
			rdjson, err := protojson.MarshalOptions{Indent: "  "}.Marshal(d)
			if err != nil {
				t.Fatal(err)
			}
			var actualJSON map[string]any
			var expectJSON map[string]any
			if err := json.Unmarshal(rdjson, &actualJSON); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal([]byte(fixture[1]), &expectJSON); err != nil {
				t.Fatal(err)
			}
			expectJSON["originalOutput"] = actualJSON["originalOutput"]
			if diff := cmp.Diff(actualJSON, expectJSON); diff != "" {
				t.Errorf("fixtures[%d] (-got, +want):\n%s", i, diff)
			}
		}
	}
}

func TestSarifParser_Suppressions(t *testing.T) {
	// SARIF 2.1.0 §3.27.23 + §3.35: a result with an "accepted" suppression
	// (Status absent or explicitly "accepted") should be skipped. Status
	// "rejected" or "underReview" must still emit diagnostics.
	sarifWithSuppression := func(suppressionsJSON string) string {
		return `{
			"version": "2.1.0",
			"runs": [
				{
					"results": [
						{
							"level": "warning",
							"locations": [
								{
									"physicalLocation": {
										"artifactLocation": {"uri": "main.tf"},
										"region": {"startLine": 1}
									}
								}
							],
							"message": {"text": "msg"},
							"ruleId": "CKV_AWS_338"` + suppressionsJSON + `
						}
					],
					"tool": {"driver": {"name": "checkov"}}
				}
			]
		}`
	}

	cases := []struct {
		name      string
		input     string
		wantCount int
	}{
		{
			name:      "suppressed: kind inSource, status absent (defaults to accepted)",
			input:     sarifWithSuppression(`, "suppressions": [{"kind": "inSource", "justification": "rationale"}]`),
			wantCount: 0,
		},
		{
			name:      "suppressed: kind inSource, status accepted",
			input:     sarifWithSuppression(`, "suppressions": [{"kind": "inSource", "status": "accepted", "justification": "rationale"}]`),
			wantCount: 0,
		},
		{
			name:      "suppressed: kind external, status absent",
			input:     sarifWithSuppression(`, "suppressions": [{"kind": "external"}]`),
			wantCount: 0,
		},
		{
			name:      "not suppressed: status rejected",
			input:     sarifWithSuppression(`, "suppressions": [{"kind": "inSource", "status": "rejected"}]`),
			wantCount: 1,
		},
		{
			name:      "not suppressed: status underReview",
			input:     sarifWithSuppression(`, "suppressions": [{"kind": "inSource", "status": "underReview"}]`),
			wantCount: 1,
		},
		{
			name:      "not suppressed: empty suppressions array",
			input:     sarifWithSuppression(`, "suppressions": []`),
			wantCount: 1,
		},
		{
			name:      "not suppressed: suppressions field absent",
			input:     sarifWithSuppression(``),
			wantCount: 1,
		},
		{
			name:      "suppressed: at least one accepted suppression among mixed",
			input:     sarifWithSuppression(`, "suppressions": [{"kind": "inSource", "status": "rejected"}, {"kind": "inSource", "status": "accepted"}]`),
			wantCount: 0,
		},
	}

	p := NewSarifParser()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diagnostics, err := p.Parse(strings.NewReader(tc.input))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if got := len(diagnostics); got != tc.wantCount {
				t.Errorf("len(diagnostics) = %d, want %d", got, tc.wantCount)
			}
		})
	}
}

func basedir() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

var fixtures = [][]string{{
	fmt.Sprintf(`{
	"$schema": "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
	"version": "2.1.0",
	"runs": [
		{
			"originalUriBaseIds": {
				"SRCROOT": {
					"uri": "file://%s"
				}
			},
			"results": [
				{
					"level": "warning",
					"locations": [
						{
							"physicalLocation": {
								"artifactLocation": {
									"uri": "src/MyClass.kt",
									"uriBaseId": "SRCROOT"
								},
								"region": {
									"startLine": 10,
									"startColumn": 5,
									"endLine": 12,
									"endColumn": 15
								}
							}
						}
					],
					"message": {
						"text": "result message"
					},
					"ruleId": "rule_id"
				}
			],
			"tool": {
				"driver": {
					"informationUri": "https://example.com",
					"name": "driver_name",
					"rules": [
						{
							"helpUri": "https://example.com",
							"id": "rule_id",
							"name": "My Rule",
							"shortDescription": {
								"text": "Rule description"
							}
						}
					]
				}
			}
		}
	]
}`, basedir()), `{
	"message": "result message",
	"location": {
		"path": "src/MyClass.kt",
		"range": {
			"start": {
				"line": 10,
				"column": 5
			},
			"end": {
				"line": 12,
				"column": 15
			}
		}
	},
	"severity": "WARNING",
	"source": {
		"name": "driver_name",
		"url": "https://example.com"
	},
	"code": {
		"value": "rule_id",
		"url": "https://example.com"
	}
}`},
	{`{
	"runs": [
		{
			"originalUriBaseIds": {
				"SRCROOT": {
					"description": {
						"text": "uri deleted root"
					}
				}
			},
			"results": [
				{
					"locations": [
						{
							"physicalLocation": {
								"artifactLocation": {
									"uri": "src/MyClass.kt",
									"uriBaseId": "SRCROOT"
								},
								"region": {
									"startLine": 10
								}
							}
						}
					],
					"message": {
						"text": "message"
					},
					"ruleId": "rule_id"
				}
			],
			"tool": {
				"driver": {
					"name": "driver_name",
					"rules": [
						{
							"id": "rule_id",
							"defaultConfiguration": {
								"level": "error"
							}
						}
					]
				}
			}
		}
	]
}`, `{
	"message": "message",
	"location": {
		"path": "src/MyClass.kt",
		"range": {
			"start": {
				"line": 10
			}
		}
	},
	"severity": "ERROR",
	"source": {
		"name": "driver_name"
	},
	"code": {
		"value": "rule_id"
	}
}`},
	{`{
	"runs": [
		{
			"results": [
				{
					"locations": [
						{
							"physicalLocation": {
								"artifactLocation": {
									"uri": "src/MyClass.kt"
								},
								"region": {
									"startLine": 10
								}
							}
						}
					],
					"fixes": [
						{
							"artifactChanges": [
								{
									"artifactLocation": {
										"uri": "src/MyClass.kt"
									},
									"replacements": [
										{
											"deletedRegion": {
												"startLine": 10,
												"startColumn": 1,
												"endColumn": 1
											},
											"insertedContent": {
												"text": "// "
											}
										}
									]
								}
							]
						}
					],
					"message": {
						"markdown": "message"
					},
					"ruleId": "rule_id"
				}
			],
			"tool": {
				"driver": {
					"name": "driver_name"
				}
			}
		}
	]
}`, `{
	"message": "message",
	"location": {
		"path": "src/MyClass.kt",
		"range": {
			"start": {
				"line": 10
			}
		}
	},
	"source": {
		"name": "driver_name"
	},
	"code": {
		"value": "rule_id"
	},
	"suggestions": [
		{
			"range": {
				"start": {
					"line": 10,
					"column": 1
				},
				"end": {
					"line": 10,
					"column": 1
				}
			},
			"text": "// "
		}
	]
}`},
	{fmt.Sprintf(`{
	"runs": [
	  {
		"originalUriBaseIds": {
			"ROOTPATH": {
			  "uri": "%s"
			}
		  },
		"tool": {
		  "driver": {
			"name": "Trivy",
			"informationUri": "https://github.com/aquasecurity/trivy",
			"fullName": "Trivy Vulnerability Scanner",
			"version": "0.15.0",
			"rules": [
			  {
				"id": "CVE-2018-14618/curl",
				"name": "OS Package Vulnerability (Alpine)",
				"shortDescription": {
				  "text": "CVE-2018-14618 Package: curl"
				},
				"fullDescription": {
				  "text": "curl: NTLM password overflow via integer overflow."
				},
				"defaultConfiguration": {
				  "level": "error"
				},
				"helpUri": "https://avd.aquasec.com/nvd/cve-2018-14618",
				"help": {
				  "text": "Vulnerability CVE-2018-14618\nSeverity: CRITICAL\n...",
				  "markdown": "**Vulnerability CVE-2018-14618**\n| Severity..."
				},
				"properties": {
				  "tags": [
					"vulnerability",
					"CRITICAL",
					"curl"
				  ],
				  "precision": "very-high"
				}
			  }
			]
		  }
		},
		"results": [
		  {
			"ruleId": "CVE-2018-14618/curl",
			"ruleIndex": 0,
			"level": "error",
			"message": {
			  "text": "curl before version 7.61.1 is..."
			},
			"locations": [{
			  "physicalLocation": {
				"artifactLocation": {
				  "uri": "knqyf263/vuln-image (alpine 3.7.1)",
				  "uriBaseId": "ROOTPATH"
				}
			  }
			}]
		  }]
	  }
	]
  }
`, basedir()), `{
	"message": "curl before version 7.61.1 is...",
	"location": {
		"path": "knqyf263/vuln-image (alpine 3.7.1)"
	},
	"severity": "ERROR",
	"source": {
		"name": "Trivy",
		"url": "https://github.com/aquasecurity/trivy"
	},
	"code": {
		"value": "CVE-2018-14618/curl",
		"url": "https://avd.aquasec.com/nvd/cve-2018-14618"
	}
}`},
	{fmt.Sprintf(`{
	"runs": [ {
		"originalUriBaseIds": {
			"ROOTPATH": {
			  "uri": "%s"
			}
		},
		"tool": {
			"driver": {
				"name": "driver_name"
			}
		},
		"results": [
			{
				"ruleId": "PY2335",
				"message": {
          "text": "Use of tainted variable 'expr' in the insecure function 'eval'."
        },
				"locations": [
					{
						"physicalLocation": {
							"artifactLocation": {
								"uri": "3-Beyond-basics/bad-eval.py"
							},
							"region": {
								"startLine": 4
							}
						}
					}
				],
				"relatedLocations": [
					{
						"message": {
							"text": "The tainted data entered the system here."
						},
						"physicalLocation": {
							"artifactLocation": {
								"uri": "3-Beyond-basics/bad-eval.py"
							},
							"region": {
								"startLine": 3
							}
						}
					}
				]
			}
			]
	  } ]
  }
`, basedir()), `{
	"message": "Use of tainted variable 'expr' in the insecure function 'eval'.",
	"location": {
		"path": "3-Beyond-basics/bad-eval.py",
    "range": {
      "start": {
        "line": 4
      }
    }
	},
	"source": {
		"name": "driver_name"
	},
	"code": {
		"value": "PY2335"
	},
  "relatedLocations": [
    {
      "message": "The tainted data entered the system here.",
			"location": {
				"path": "3-Beyond-basics/bad-eval.py",
				"range": {
					"start": {
						"line": 3
					}
				}
			}
    }
  ]
}`},
}
