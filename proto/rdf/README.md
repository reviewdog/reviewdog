---
title: Reviewdog Diagnostic Format
date: 2020-06-15
author: haya14busa
status: Proposed / Experimental
---

# Status

This document proposes Reviewdog Diagnostic Format and it's still
in experimental stage.

Any review, suggestion, feedback, criticism, and comments from anyone is very
much welcome. Please leave comments in Pull Request ([#629](https://github.com/reviewdog/reviewdog/pull/629)),
in issue [#628](https://github.com/reviewdog/reviewdog/issues/628) or
file [an issue](https://github.com/reviewdog/reviewdog/issues).

The document and the actual definition are currently under the
https://github.com/reviewdog/reviewdog repository, but we may create a separate
repository once it's reviewed and stabilized.

# Reviewdog Diagnostic Format (RDFormat)

Reviewdog Diagnostic Format defines standard machine-readable message
structures which represent a result of diagnostic tool such as a compiler or a
linter.

The idea behind the Reviewdog Diagnostic Format is to standardize
the protocol for how diagnostic tools (e.g. compilers, linters, etc..) and
development tools (e.g. editors, reviewdog, code review API etc..) communicate.

See [reviewdog.proto](reviewdog.proto) for the actual definition.
[JSON Schema](./jsonschema) is available as well.

## Wire formats of Reviewdog Diagnostic Format.

RDFormat uses [Protocol Buffer](https://developers.google.com/protocol-buffers) to
define the message structure, but the recommended wire format is JSON considering
it's widely used and easy to support both from diagnostic tools and development
tools.

### **rdjsonl**
JSON Lines (http://jsonlines.org/) of the [`Diagnostic`](reviewdog.proto) message ([JSON Schema](./jsonschema/Diagnostic.jsonschema)).

Example:
```json
{"message": "<msg>", "location": {"path": "<file path>", "range": {"start": {"line": 14, "column": 15}}}, "severity": "ERROR"}
{"message": "<msg>", "location": {"path": "<file path>", "range": {"start": {"line": 14, "column": 15}, "end": {"line": 14, "column": 18}}}, "suggestions": [{"range": {"start": {"line": 14, "column": 15}, "end": {"line": 14, "column": 18}}, "text": "<replacement text>"}], "severity": "WARNING"}
...
```

### **rdjson**
JSON format of the [`DiagnosticResult`](reviewdog.proto) message ([JSON Schema](./jsonschema/DiagnosticResult.jsonschema)).

Example:
```json
{
  "source": {
    "name": "super lint",
    "url": "https://example.com/url/to/super-lint"
  },
  "severity": "WARNING",
  "diagnostics": [
    {
      "message": "<msg>",
      "location": {
        "path": "<file path>",
        "range": {
          "start": {
            "line": 14,
            "column": 15
          }
        }
      },
      "severity": "ERROR",
      "code": {
        "value": "RULE1",
        "url": "https://example.com/url/to/super-lint/RULE1"
      }
    },
    {
      "message": "<msg>",
      "location": {
        "path": "<file path>",
        "range": {
          "start": {
            "line": 14,
            "column": 15
          },
          "end": {
            "line": 14,
            "column": 18
          }
        }
      },
      "suggestions": [
        {
          "range": {
            "start": {
              "line": 14,
              "column": 15
            },
            "end": {
              "line": 14,
              "column": 18
            }
          },
          "text": "<replacement text>"
        }
      ],
      "severity": "WARNING"
    }
  ]
}
```

## Background: Still No Good Standard Diagnostic Format Out There in 2020

Update: Found *The Static Analysis Results Interchange Format (SARIF)* as a
potential good standard format.

As of writing (2020), most diagnostic tools such as linters or compilers output
results with their own format. Some tools support machine-readable structured
format like their own JSON format, and other tools just support unstructured
format (e.g. `/path/to/file:<line>:<column>: <message>`).

The fact that there are no standard formats for diagnostic tools' output makes
it hard to integrate diagnostic tools with development tools such as editors or
automated code review tools/services.

[reviewdog](https://github.com/reviewdog/reviewdog) resolves the above problem
by introducing [errorformat](https://github.com/reviewdog/errorformat) to
support unstructured output and checkstyle XML format as structured output.
It works great so far and reviewdog can support arbitrary diagnostic tools
regardless of programming languages. However, these solutions doesn't solve
everything.

### *errorformat*
[errorformat](https://github.com/reviewdog/errorformat)

Problems:
- No support for diagnostics for code range. It only supports start position.
- No support for code suggestions (also known as auto-correct or fix).
- It's hard to write errorformat for complicated output.

### *checkstyle XML format*
[checkstyle](https://checkstyle.sourceforge.io/)

Problems:
- No support for diagnostics for code range. It only supports start position.
- No support for code suggestions (also known as auto-correct or fix).
- It's ..... XML. It's true that some diagnostic tools support checkstyle
format, but not everyone wants to support it.
- The checkstyle itself is actually a diagnostic tool for Java and its
  output format is actually not well-documented and not meant to be
  used as generic format. Some linters just happens to use the same format(?).

## Background: Altenatives

There are altenative solutions out there (which are not used by reviewdog) as
well.

### The Static Analysis Results Interchange Format (SARIF)
[The Static Analysis Results Interchange Format (SARIF)](https://sarifweb.azurewebsites.net/)
has been approved as an OASIS standard.

Although, there are not many usages of SARIF as of writing (2020 July, 21),
it can be good standard format.
A promising usage example is [GitHub Code Scanning](https://docs.github.com/en/github/finding-security-vulnerabilities-and-errors-in-your-code/about-code-scanning#about-third-party-code-scanning-tools)
(beta), which uses SARIF to support third party code scanning tools.
Other examples: [spotbugs](https://github.com/spotbugs/discuss/issues/95).

Problems:
- No stream output support and static analysis tools cannot output each diagnostic result one by one.
- `columnKind` doesn't support byte count. https://github.com/oasis-tcs/sarif-spec/issues/466
- The spec is too big and complex ([SARIF v2.1.0 PDF](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.pdf) is 227 pages!)
  for developer tools as consumer of SARIF (e.g.  reviewdog). Probably most
  tools end up with supporting SARIF partially.  GitHub Code Scanning feature
  actually doesn't support a whole spec
  ([doc](https://docs.github.com/en/github/finding-security-vulnerabilities-and-errors-in-your-code/sarif-support-for-code-scanning))
  for example.
- The spec is too big and complex for static analysis tools as provider of
  SARIF. They can just support partial and minimum SARIF support as result
  output format but it's still not simple and the output still needs to pass
  SARIF validatiotor.
- Not all languages have good tools to generate code from JSON Schema.
  To create Go SARIF package [haya14busa/go-sarif](https://github.com/haya14busa/go-sarif),
  I needed to try 3+ [Go](https://github.com/atombender/go-jsonschema) [JSON Schema](https://github.com/idubinskiy/schematyper)
  [Code Generator](https://github.com/aaharu/schemarshal)
  tools but all of them didn't work for the complex SARIF JSON Schema.
  I ended up using [quicktype](https://github.com/quicktype/quicktype) and it
  worked but I still needed to send [a Pull Request](https://github.com/quicktype/quicktype/pull/1513)...
- SARIF SDK and related tools are written in C# (and TypeScript), which means we need dotnet runtime.
  SARIF is general and standard format while the related tools requires dotnet runtime.

There are some problems as above but SARIF should be still good to support
considering it has been already approved as an OASIS standard and GitHub Code
Scanning uses it.
Reviewdog Diagnostic Format can be used as simpler format and we can create
converters between RD Format and SARIF.

### *Problem Matcher*
[VSCode](https://vscode-docs.readthedocs.io/en/stable/editor/tasks/#defining-a-problem-matcher)
and [GitHub Actions](https://github.com/actions/toolkit/blob/master/docs/problem-matchers.md)
uses [Problem Matcher](https://github.com/actions/toolkit/blob/master/docs/problem-matchers.md)
to support arbitrary diagnostic tools. It's similar to errorformat, but it uses regex.

Problems:
- No support for code suggestions (also known as auto-correct or fix).
- Output format of matched results are undocumented and it seems to be used internally in VSCode and GitHub Actions.
- It's hard to write problem matchers for complicated output.

### *Language Server Protocol (LSP)*
[Language Server Protocol Specification](https://microsoft.github.io/language-server-protocol/specifications/specification-current/)

LSP supports [Diagnostic](https://microsoft.github.io/language-server-protocol/specifications/specification-current/#diagnostic)
to represents a diagnostic, such as a compiler error or warning.
It's great for editor integration and is widely used these days as well.
RDFormat message is actually inspired by LSP Diagnostic message too.

Problems:
- LSP and the Diagnostic message is basically per one file. It's not always
  suited to be used as diagnostic tools output because they often need to
  report diagnostic results for multiple files and outputing json per file does
  not make very much sense.
- LSP's Diagnostic message doesn't have code suggestions (code action) data.
  Code action have data about associated diagnostic on the contrary and the
  code action message itself doesn't contain text edit data too, so LSP's
  messages are not suited to represent a diagnosis result with suggested fix.
- Unnatural position representation: Position in LSP are zero-based and
  character offset is based on [UTF-16 code units](https://github.com/microsoft/language-server-protocol/issues/376).
  These are not widely used by diagnostic tools, development tools nor code
  review API such as GitHub, GitLab and Gerrit....
  In addition, UTF-8 is defact-standard of text file encoding as well these days.

## Reviewdog Diagnostic Format Concept
Again, the idea behind the Reviewdog Diagnostic Format (RDFormat) is to
standardize the protocol for how diagnostic tools (e.g. compilers, linters,
etc..) and development tools (e.g. editors, reviewdog, code review API etc..)
communicate.

RDFormat should support major use cases from representing diagnostic results to
apply suggested fix in general way and should be easily supported by diagnostic
tools and development tools regardless of their programming languages.

[![Reviewdog Diagnostic Format Concept](https://user-images.githubusercontent.com/3797062/87955046-2b8b6300-cae8-11ea-983f-6554e2aeb8f2.png)](https://docs.google.com/drawings/d/15GZu5Iq6wukFtrpy91srQO_ry1iFQUisVAJd_yEprLc/edit?usp=sharing)

### Diagnostic tools' RDFormat Support
Ideally, diagnostic tools themselves should support outputing their results as
RDFormat compliant format, but not all tools does support RDFormat especially
in early stage. But we can still introduce RDFormat by supporting RDFormat with
errorformat for most diagnostic tools. Also, we can write a converter and add
RPD support in diagnostic tools incrementally.

### Consumer: reviewdog
*Not implemented yet*

reviewdog can support RDFormat and consume `rdjsonl`/`rdjson` as structured input
of diagnostic tools.
It also makes it possible to support (1) a diagnostic to code range and (2)
code suggestions (auto-correction) if a reporter supports them (e.g.
github-pr-review, gitlab-mr-discussion and local reporter).

As for suggestion support with local reporter, reviewdog should be able to
apply suggestions only in diff for example.

### Consumer: Editor & Language Server Protocol
*Not implemented yet*

It's going to be easier for editors to support arbitrary diagnostic tools by
using RDFormat. Language Server can also use RDFormat and it's easy to convert RDFormat
message to LSP Diagnostic and/or Code Action message.

One possible more concrete idea is to extend
[efm-langserver](https://github.com/mattn/efm-langserver) to support RDFormat
message as input.
efm-langserver currently uses
[errorformat](https://github.com/reviewdog/errorformat) to support diagnostic
tools generally, but not all tools' output can be easily parsed with
errorformat and errorformat lacks some features like diagnostics for code range.
It should be able to support code action to apply suggested fix as well.

### Consumer: Reviewdog Diagnostic Formatter (RDFormatter)
*Not implemented yet*

There are many diagnostic output formats (report formats) and each diagnostic
tool implements them on their own. e.g. [eslint](https://eslint.org/docs/user-guide/formatters)
support more than 10 formats like stylish, compact, codeframe, html, etc...
Users may want to use a certain format for every diagnostic tools they use, but 
not all tools support their desired format. It takes time to implement many
formats for each tool and it's actually not worth doing it for most of the
cases, IMO.

Reviewdog Diagnostic Formatter should support formating of diagnostic
results based on RDfFormat. Then, diagnostic tools can focus on improving
diagnostic feature and let the formatter to format the results.

RDFormatter should be provided both as CLI and as libraries.
The CLI can take RDFormat messages as input and output formatted results. The CLI
should be especially useful to build special format like custom html to
generate report pages independing on diagnostic tools nor their implementation
languages. However, many diagnostic tools and users should not always want to
depend on the CLI, so providing libraries for their implementation languages
should be useful to format results natively by each diagnostic tool.

## Open Questions
- Protocol Version Representation and Backward/Future Compatibility
  - Should we add version or some capability data in RD Format?
  - RD Format should be stable, but there are still a possibility to extend it with
    backward incompatible way. e.g. We **may** want to add byte offset field in
    Position message as an alternative of line and column.
