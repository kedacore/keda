package gotest

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jstemmer/go-junit-report/v2/gtr"
	"github.com/jstemmer/go-junit-report/v2/parser/gotest/internal/collector"
)

const (
	globalID = 0
)

// reportBuilder helps build a test Report from a collection of events.
//
// The reportBuilder delegates to the packageBuilder for creating packages from
// basic test events, but keeps track of build errors itself. The reportBuilder
// is also responsible for generating unique test id's.
//
// Test output is collected by the output collector, which also keeps track of
// the currently active test so output is automatically associated with the
// correct test.
type reportBuilder struct {
	packageBuilders map[string]*packageBuilder
	buildErrors     map[int]gtr.Error

	nextID   int               // next free unused id
	output   *collector.Output // output collected for each id
	packages []gtr.Package     // completed packages

	// options
	packageName   string
	subtestMode   SubtestMode
	timestampFunc func() time.Time
}

// newReportBuilder creates a new reportBuilder.
func newReportBuilder() *reportBuilder {
	return &reportBuilder{
		packageBuilders: make(map[string]*packageBuilder),
		buildErrors:     make(map[int]gtr.Error),
		nextID:          1,
		output:          collector.New(),
		timestampFunc:   time.Now,
	}
}

// getPackageBuilder returns the packageBuilder for the given packageName. If
// no packageBuilder exists for the given package, a new one is created.
func (b *reportBuilder) getPackageBuilder(packageName string) *packageBuilder {
	pb, ok := b.packageBuilders[packageName]
	if !ok {
		output := b.output
		if packageName != "" {
			output = collector.New()
		}
		pb = newPackageBuilder(b.generateID, output)
		b.packageBuilders[packageName] = pb
	}
	return pb
}

// ProcessEvent takes a test event and adds it to the report.
func (b *reportBuilder) ProcessEvent(ev Event) {
	switch ev.Type {
	case "run_test":
		b.getPackageBuilder(ev.Package).CreateTest(ev.Name)
	case "pause_test":
		b.getPackageBuilder(ev.Package).PauseTest(ev.Name)
	case "cont_test":
		b.getPackageBuilder(ev.Package).ContinueTest(ev.Name)
	case "end_test":
		b.getPackageBuilder(ev.Package).EndTest(ev.Name, ev.Result, ev.Duration, ev.Indent)
	case "run_benchmark":
		b.getPackageBuilder(ev.Package).CreateTest(ev.Name)
	case "benchmark":
		b.getPackageBuilder(ev.Package).BenchmarkResult(ev.Name, ev.Iterations, ev.NsPerOp, ev.MBPerSec, ev.BytesPerOp, ev.AllocsPerOp)
	case "end_benchmark":
		b.getPackageBuilder(ev.Package).EndTest(ev.Name, ev.Result, 0, 0)
	case "status":
		b.getPackageBuilder(ev.Package).End()
	case "summary":
		// The summary marks the end of a package. We can now create the actual
		// package from all the events we've processed so far for this package.
		b.packages = append(b.packages, b.CreatePackage(ev.Package, ev.Name, ev.Result, ev.Duration, ev.Data))
	case "coverage":
		b.getPackageBuilder(ev.Package).Coverage(ev.CovPct, ev.CovPackages)
	case "build_output":
		b.CreateBuildError(ev.Name)
	case "output":
		if ev.Package != "" {
			b.getPackageBuilder(ev.Package).Output(ev.Data)
		} else {
			b.output.Append(ev.Data)
		}
	default:
		// This shouldn't happen, but just in case print a warning and ignore
		// this event.
		fmt.Printf("reportBuilder: unhandled event type: %v\n", ev.Type)
	}
}

// newID returns a new unique id.
func (b *reportBuilder) generateID() int {
	id := b.nextID
	b.nextID++
	return id
}

// Build returns the new Report containing all the tests, build errors and
// their output created from the processed events.
func (b *reportBuilder) Build() gtr.Report {
	// Create packages for any leftover package builders.
	for name, pb := range b.packageBuilders {
		if pb.IsEmpty() {
			continue
		}
		b.packages = append(b.packages, b.CreatePackage(name, b.packageName, "", 0, ""))
	}

	// Create packages for any leftover build errors.
	for _, buildErr := range b.buildErrors {
		b.packages = append(b.packages, b.CreatePackage("", buildErr.Name, "", 0, ""))
	}
	return gtr.Report{Packages: b.packages}
}

// CreateBuildError creates a new build error and marks it as active.
func (b *reportBuilder) CreateBuildError(packageName string) {
	id := b.generateID()
	b.output.SetActiveID(id)
	b.buildErrors[id] = gtr.Error{ID: id, Name: packageName}
}

// CreatePackage returns a new package containing all the build errors, output,
// tests and benchmarks created so far. The optional packageName is used to
// find the correct reportBuilder. The newPackageName is the actual package
// name that will be given to the returned package, which should be used in
// case the packageName was unknown until this point.
func (b *reportBuilder) CreatePackage(packageName, newPackageName, result string, duration time.Duration, data string) gtr.Package {
	pkg := gtr.Package{
		Name:      newPackageName,
		Duration:  duration,
		Timestamp: b.timestampFunc(),
	}

	// First check if this package contained a build error. If that's the case,
	// we won't find any tests in this package.
	for id, buildErr := range b.buildErrors {
		if buildErr.Name == newPackageName || strings.TrimSuffix(buildErr.Name, "_test") == newPackageName {
			pkg.BuildError = buildErr
			pkg.BuildError.ID = id
			pkg.BuildError.Duration = duration
			pkg.BuildError.Cause = data
			pkg.BuildError.Output = b.output.Get(id)

			delete(b.buildErrors, id)
			b.output.SetActiveID(0)
			return pkg
		}
	}

	// Get the packageBuilder for this package and make sure it's deleted, so
	// future events for this package will use a new packageBuilder.
	pb := b.getPackageBuilder(packageName)
	delete(b.packageBuilders, packageName)
	pb.output.SetActiveID(0)

	// If the packageBuilder is empty, we never received any events for this
	// package so there's no need to continue.
	if pb.IsEmpty() {
		// However, we should at least report an error if the result says we
		// failed.
		if parseResult(result) == gtr.Fail {
			pkg.RunError = gtr.Error{
				Name: newPackageName,
			}
		}
		return pkg
	}

	// If we've collected output, but there were no tests, then this package
	// had a runtime error or it simply didn't have any tests.
	if pb.output.Contains(globalID) && len(pb.tests) == 0 {
		if parseResult(result) == gtr.Fail {
			pkg.RunError = gtr.Error{
				Name:   newPackageName,
				Output: pb.output.Get(globalID),
			}
		} else {
			pkg.Output = pb.output.Get(globalID)
		}
		pb.output.Clear(globalID)
		return pkg
	}

	// If the summary result says we failed, but there were no failing tests
	// then something else must have failed.
	if parseResult(result) == gtr.Fail && len(pb.tests) > 0 && !pb.containsFailures() {
		pkg.RunError = gtr.Error{
			Name:   newPackageName,
			Output: pb.output.Get(globalID),
		}
		pb.output.Clear(globalID)
	}

	// Collect tests for this package
	var tests []gtr.Test
	for id, t := range pb.tests {
		if pb.isParent(id) {
			if b.subtestMode == IgnoreParentResults {
				t.Result = gtr.Pass
			} else if b.subtestMode == ExcludeParents {
				pb.output.Merge(id, globalID)
				continue
			}
		}
		t.Output = pb.output.Get(id)
		tests = append(tests, t)
	}
	tests = groupBenchmarksByName(tests, b.output)

	// Sort packages by id to ensure we maintain insertion order.
	sort.Slice(tests, func(i, j int) bool {
		return tests[i].ID < tests[j].ID
	})

	pkg.Tests = groupBenchmarksByName(tests, pb.output)
	pkg.Coverage = pb.coverage
	pkg.Output = pb.output.Get(globalID)
	pb.output.Clear(globalID)
	return pkg
}

// parseResult returns a gtr.Result for the given result string r.
func parseResult(r string) gtr.Result {
	switch r {
	case "PASS":
		return gtr.Pass
	case "FAIL":
		return gtr.Fail
	case "SKIP":
		return gtr.Skip
	case "BENCH":
		return gtr.Pass
	default:
		return gtr.Unknown
	}
}

// groupBenchmarksByName groups tests with the Benchmark prefix if they have
// the same name and combines their output.
func groupBenchmarksByName(tests []gtr.Test, output *collector.Output) []gtr.Test {
	if len(tests) == 0 {
		return nil
	}

	var grouped []gtr.Test
	byName := make(map[string][]gtr.Test)
	for _, test := range tests {
		if !strings.HasPrefix(test.Name, "Benchmark") {
			// If this test is not a benchmark, we won't group it by name but
			// just add it to the final result.
			grouped = append(grouped, test)
			continue
		}
		if _, ok := byName[test.Name]; !ok {
			grouped = append(grouped, gtr.NewTest(test.ID, test.Name))
		}
		byName[test.Name] = append(byName[test.Name], test)
	}

	for i, group := range grouped {
		if !strings.HasPrefix(group.Name, "Benchmark") {
			continue
		}
		var (
			ids   []int
			total Benchmark
			count int
		)
		for _, test := range byName[group.Name] {
			ids = append(ids, test.ID)
			if test.Result != gtr.Pass {
				continue
			}

			if bench, ok := GetBenchmarkData(test); ok {
				total.Iterations += bench.Iterations
				total.NsPerOp += bench.NsPerOp
				total.MBPerSec += bench.MBPerSec
				total.BytesPerOp += bench.BytesPerOp
				total.AllocsPerOp += bench.AllocsPerOp
				count++
			}
		}

		group.Duration = combinedDuration(byName[group.Name])
		group.Result = groupResults(byName[group.Name])
		group.Output = output.GetAll(ids...)
		if count > 0 {
			total.Iterations /= int64(count)
			total.NsPerOp /= float64(count)
			total.MBPerSec /= float64(count)
			total.BytesPerOp /= int64(count)
			total.AllocsPerOp /= int64(count)
			SetBenchmarkData(&group, total)
		}
		grouped[i] = group
	}
	return grouped
}

// combinedDuration returns the sum of the durations of the given tests.
func combinedDuration(tests []gtr.Test) time.Duration {
	var total time.Duration
	for _, test := range tests {
		total += test.Duration
	}
	return total
}

// groupResults returns the result we should use for a collection of tests.
func groupResults(tests []gtr.Test) gtr.Result {
	var result gtr.Result
	for _, test := range tests {
		if test.Result == gtr.Fail {
			return gtr.Fail
		}
		if result != gtr.Pass {
			result = test.Result
		}
	}
	return result
}

// packageBuilder helps build a gtr.Package from a collection of test events.
type packageBuilder struct {
	generateID func() int
	output     *collector.Output

	tests     map[int]gtr.Test
	parentIDs map[int]struct{} // set of test id's that contain subtests
	coverage  float64          // coverage percentage
}

// newPackageBuilder creates a new packageBuilder. New tests will be assigned
// an ID returned by the generateID function. The activeIDSetter is called to
// set or reset the active test id.
func newPackageBuilder(generateID func() int, output *collector.Output) *packageBuilder {
	return &packageBuilder{
		generateID: generateID,
		output:     output,
		tests:      make(map[int]gtr.Test),
		parentIDs:  make(map[int]struct{}),
	}
}

// IsEmpty returns true if this package builder does not have any tests and has
// not collected any global output.
func (b packageBuilder) IsEmpty() bool {
	return len(b.tests) == 0 && !b.output.Contains(0)
}

// CreateTest adds a test with the given name to the package, marks it as
// active and returns its generated id.
func (b *packageBuilder) CreateTest(name string) int {
	if parentID, ok := b.findTestParentID(name); ok {
		b.parentIDs[parentID] = struct{}{}
	}
	id := b.generateID()
	b.output.SetActiveID(id)
	b.tests[id] = gtr.NewTest(id, name)
	return id
}

// PauseTest marks the test with the given name no longer active. Any results
// or output added to the package after calling PauseTest will no longer be
// associated with this test.
func (b *packageBuilder) PauseTest(name string) {
	b.output.SetActiveID(0)
}

// ContinueTest finds the test with the given name and marks it as active. If
// more than one test exist with this name, the most recently created test will
// be used.
func (b *packageBuilder) ContinueTest(name string) {
	id, _ := b.findTest(name)
	b.output.SetActiveID(id)
}

// EndTest finds the test with the given name, sets the result, duration and
// level. If more than one test exists with this name, the most recently
// created test will be used. If no test exists with this name, a new test is
// created. The test is then marked as no longer active.
func (b *packageBuilder) EndTest(name, result string, duration time.Duration, level int) {
	id, ok := b.findTest(name)
	if !ok {
		// test did not exist, create one
		// TODO: Likely reason is that the user ran go test without the -v
		// flag, should we report this somewhere?
		id = b.CreateTest(name)
	}

	t := b.tests[id]
	t.Result = parseResult(result)
	t.Duration = duration
	t.Level = level
	b.tests[id] = t
	b.output.SetActiveID(0)
}

// End resets the active test.
func (b *packageBuilder) End() {
	b.output.SetActiveID(0)
}

// BenchmarkResult updates an existing or adds a new test with the given
// results and marks it as active. If an existing test with this name exists
// but without result, then that one is updated. Otherwise a new one is added
// to the report.
func (b *packageBuilder) BenchmarkResult(name string, iterations int64, nsPerOp, mbPerSec float64, bytesPerOp, allocsPerOp int64) {
	id, ok := b.findTest(name)
	if !ok || b.tests[id].Result != gtr.Unknown {
		id = b.CreateTest(name)
	}
	b.output.SetActiveID(id)

	benchmark := Benchmark{iterations, nsPerOp, mbPerSec, bytesPerOp, allocsPerOp}
	test := gtr.NewTest(id, name)
	test.Result = gtr.Pass
	test.Duration = benchmark.ApproximateDuration()
	SetBenchmarkData(&test, benchmark)
	b.tests[id] = test
}

// Coverage sets the code coverage percentage.
func (b *packageBuilder) Coverage(pct float64, packages []string) {
	b.coverage = pct
}

// Output appends data to the output of this package.
func (b *packageBuilder) Output(data string) {
	b.output.Append(data)
}

// findTest returns the id of the most recently created test with the given
// name if it exists.
func (b *packageBuilder) findTest(name string) (int, bool) {
	var maxid int
	for id, test := range b.tests {
		if maxid < id && test.Name == name {
			maxid = id
		}
	}
	return maxid, maxid > 0
}

// findTestParentID searches the existing tests in this package for a parent of
// the test with the given name, and returns its id if one is found.
func (b *packageBuilder) findTestParentID(name string) (int, bool) {
	parent := dropLastSegment(name)
	for parent != "" {
		if id, ok := b.findTest(parent); ok {
			return id, true
		}
		parent = dropLastSegment(parent)
	}
	return 0, false
}

// isParent returns true if the test with the given id has sub tests.
func (b *packageBuilder) isParent(id int) bool {
	_, ok := b.parentIDs[id]
	return ok
}

// dropLastSegment strips the last `/` and everything following it from the
// given name. If no `/` was found, the empty string is returned.
func dropLastSegment(name string) string {
	if idx := strings.LastIndexByte(name, '/'); idx >= 0 {
		return name[:idx]
	}
	return ""
}

// containsFailures return true if this package contains at least one failing
// test or a test with an unknown result.
func (b *packageBuilder) containsFailures() bool {
	for _, test := range b.tests {
		if test.Result == gtr.Fail || test.Result == gtr.Unknown {
			return true
		}
	}
	return false
}
