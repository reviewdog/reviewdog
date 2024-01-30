package parser

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/reviewdog/reviewdog/service/serviceutil"
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

func basedir() string {
	root, err := serviceutil.GetGitRoot()
	if err != nil {
		panic(err)
	}
	return root
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
					"description": "uri deleted root"
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
}
