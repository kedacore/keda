// Package junit defines a JUnit XML report and includes convenience methods
// for working with these reports.
package junit

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jstemmer/go-junit-report/v2/gtr"
)

// Testsuites is a collection of JUnit testsuites.
type Testsuites struct {
	XMLName xml.Name `xml:"testsuites"`

	Name     string `xml:"name,attr,omitempty"`
	Time     string `xml:"time,attr,omitempty"` // total duration in seconds
	Tests    int    `xml:"tests,attr,omitempty"`
	Errors   int    `xml:"errors,attr,omitempty"`
	Failures int    `xml:"failures,attr,omitempty"`
	Skipped  int    `xml:"skipped,attr,omitempty"`
	Disabled int    `xml:"disabled,attr,omitempty"`

	Suites []Testsuite `xml:"testsuite,omitempty"`
}

// AddSuite adds a Testsuite and updates this testssuites' totals.
func (t *Testsuites) AddSuite(ts Testsuite) {
	t.Suites = append(t.Suites, ts)
	t.Tests += ts.Tests
	t.Errors += ts.Errors
	t.Failures += ts.Failures
	t.Skipped += ts.Skipped
	t.Disabled += ts.Disabled
}

// WriteXML writes the XML representation of Testsuites t to writer w.
func (t *Testsuites) WriteXML(w io.Writer) error {
	enc := xml.NewEncoder(w)
	enc.Indent("", "\t")
	if err := enc.Encode(t); err != nil {
		return err
	}
	if err := enc.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "\n")
	return err
}

// Testsuite is a single JUnit testsuite containing testcases.
type Testsuite struct {
	// required attributes
	Name     string `xml:"name,attr"`
	Tests    int    `xml:"tests,attr"`
	Failures int    `xml:"failures,attr"`
	Errors   int    `xml:"errors,attr"`
	ID       int    `xml:"id,attr"`

	// optional attributes
	Disabled  int    `xml:"disabled,attr,omitempty"`
	Hostname  string `xml:"hostname,attr,omitempty"`
	Package   string `xml:"package,attr,omitempty"`
	Skipped   int    `xml:"skipped,attr,omitempty"`
	Time      string `xml:"time,attr"`                // duration in seconds
	Timestamp string `xml:"timestamp,attr,omitempty"` // date and time in ISO8601
	File      string `xml:"file,attr,omitempty"`

	Properties *[]Property `xml:"properties>property,omitempty"`
	Testcases  []Testcase  `xml:"testcase,omitempty"`
	SystemOut  *Output     `xml:"system-out,omitempty"`
	SystemErr  *Output     `xml:"system-err,omitempty"`
}

// AddProperty adds a property with the given name and value to this Testsuite.
func (t *Testsuite) AddProperty(name, value string) {
	prop := Property{Name: name, Value: value}
	if t.Properties == nil {
		t.Properties = &[]Property{prop}
		return
	}
	props := append(*t.Properties, prop)
	t.Properties = &props
}

// AddTestcase adds Testcase tc to this Testsuite.
func (t *Testsuite) AddTestcase(tc Testcase) {
	t.Testcases = append(t.Testcases, tc)
	t.Tests++

	if tc.Error != nil {
		t.Errors++
	}

	if tc.Failure != nil {
		t.Failures++
	}

	if tc.Skipped != nil {
		t.Skipped++
	}
}

// SetTimestamp sets the timestamp in this Testsuite.
func (t *Testsuite) SetTimestamp(timestamp time.Time) {
	t.Timestamp = timestamp.Format(time.RFC3339)
}

// Testcase represents a single test with its results.
type Testcase struct {
	// required attributes
	Name      string `xml:"name,attr"`
	Classname string `xml:"classname,attr"`

	// optional attributes
	Time   string `xml:"time,attr,omitempty"` // duration in seconds
	Status string `xml:"status,attr,omitempty"`

	Skipped   *Result `xml:"skipped,omitempty"`
	Error     *Result `xml:"error,omitempty"`
	Failure   *Result `xml:"failure,omitempty"`
	SystemOut *Output `xml:"system-out,omitempty"`
	SystemErr *Output `xml:"system-err,omitempty"`
}

// Property represents a key/value pair.
type Property struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// Result represents the result of a single test.
type Result struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr,omitempty"`
	Data    string `xml:",cdata"`
}

// Output represents output written to stdout or sderr.
type Output struct {
	Data string `xml:",cdata"`
}

// CreateFromReport creates a JUnit representation of the given gtr.Report.
func CreateFromReport(report gtr.Report, hostname string) Testsuites {
	var suites Testsuites
	for _, pkg := range report.Packages {
		var duration time.Duration
		suite := Testsuite{
			Name:     pkg.Name,
			Hostname: hostname,
			ID:       len(suites.Suites),
		}

		if !pkg.Timestamp.IsZero() {
			suite.SetTimestamp(pkg.Timestamp)
		}

		for _, p := range pkg.Properties {
			suite.AddProperty(p.Name, p.Value)
		}

		if len(pkg.Output) > 0 {
			suite.SystemOut = &Output{Data: formatOutput(pkg.Output)}
		}

		if pkg.Coverage > 0 {
			suite.AddProperty("coverage.statements.pct", fmt.Sprintf("%.2f", pkg.Coverage))
		}

		for _, test := range pkg.Tests {
			duration += test.Duration
			suite.AddTestcase(createTestcaseForTest(pkg.Name, test))
		}

		// JUnit doesn't have a good way of dealing with build or runtime
		// errors that happen before a test has started, so we create a single
		// failing test that contains the build error details.
		if pkg.BuildError.Name != "" {
			tc := Testcase{
				Classname: pkg.BuildError.Name,
				Name:      pkg.BuildError.Cause,
				Time:      formatDuration(0),
				Error: &Result{
					Message: "Build error",
					Data:    strings.Join(pkg.BuildError.Output, "\n"),
				},
			}
			suite.AddTestcase(tc)
		}

		if pkg.RunError.Name != "" {
			tc := Testcase{
				Classname: pkg.RunError.Name,
				Name:      "Failure",
				Time:      formatDuration(0),
				Error: &Result{
					Message: "Runtime error",
					Data:    strings.Join(pkg.RunError.Output, "\n"),
				},
			}
			suite.AddTestcase(tc)
		}

		if (pkg.Duration) == 0 {
			suite.Time = formatDuration(duration)
		} else {
			suite.Time = formatDuration(pkg.Duration)
		}
		suites.AddSuite(suite)
	}
	return suites
}

func createTestcaseForTest(pkgName string, test gtr.Test) Testcase {
	tc := Testcase{
		Classname: pkgName,
		Name:      test.Name,
		Time:      formatDuration(test.Duration),
	}

	if test.Result == gtr.Fail {
		tc.Failure = &Result{
			Message: "Failed",
			Data:    formatOutput(test.Output),
		}
	} else if test.Result == gtr.Skip {
		tc.Skipped = &Result{
			Message: "Skipped",
			Data:    formatOutput(test.Output),
		}
	} else if test.Result == gtr.Unknown {
		tc.Error = &Result{
			Message: "No test result found",
			Data:    formatOutput(test.Output),
		}
	} else if len(test.Output) > 0 {
		tc.SystemOut = &Output{Data: formatOutput(test.Output)}
	}
	return tc
}

// formatDuration returns the JUnit string representation of the given
// duration.
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.3f", d.Seconds())
}

// formatOutput combines the lines from the given output into a single string.
func formatOutput(output []string) string {
	return escapeIllegalChars(strings.Join(output, "\n"))
}

func escapeIllegalChars(str string) string {
	return strings.Map(func(r rune) rune {
		if isInCharacterRange(r) {
			return r
		}
		return '\uFFFD'
	}, str)
}

// Decide whether the given rune is in the XML Character Range, per
// the Char production of https://www.xml.com/axml/testaxml.htm,
// Section 2.2 Characters.
// From: encoding/xml/xml.go
func isInCharacterRange(r rune) (inrange bool) {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}
