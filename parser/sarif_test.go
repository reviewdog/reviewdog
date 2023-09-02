package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

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
		for _, d := range diagnostics {
			rdjson, _ := protojson.MarshalOptions{Indent: "  "}.Marshal(d)
			var actualJson interface{}
			var expectJson interface{}
			json.Unmarshal([]byte(rdjson), &actualJson)
			json.Unmarshal([]byte(fixture[1]), &expectJson)
			if !reflect.DeepEqual(actualJson, expectJson) {
				var out bytes.Buffer
				json.Indent(&out, rdjson, "", "\t")
				actual := out.String()
				expect := fixture[1]
				t.Errorf("actual(%v):\n%v\n---\nexpect(%v):\n%v", i, actual, i, expect)
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
	},
	"originalOutput": "src/MyClass.kt:10:5: warning: result message (rule_id)"
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
	},
	"originalOutput": "src/MyClass.kt:10:1: error: message (rule_id)"
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
	],
	"originalOutput": "src/MyClass.kt:10:1: : message (rule_id)"
}`},
}
