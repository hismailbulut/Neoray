// This package calculates the executing time of the functions.
package measurer

import (
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	tw "github.com/olekukonko/tablewriter"
)

type function_measure struct {
	calls int
	time  time.Duration
	max   time.Duration
}

var (
	averages map[string]function_measure
	mutex    sync.Mutex
)

// Initializes package, only call once at the beginning.
func Init() {
	averages = make(map[string]function_measure)
}

// This function stores current time and returns a function that does the actual
// calculation. You must call the returned function at the end of your function.
// You can simply add this line to the beginning:
// defer measurer.Measure()()
// The double pharanteses are important. Also you can give it a name.
// defer measurer.Measure()("my_function_2")
func Measure() func(custom ...string) {
	now := time.Now()
	return func(custom ...string) {
		name := "Unrecognized"
		if len(custom) > 0 {
			name = custom[0]
		} else {
			pc, _, _, ok := runtime.Caller(1)
			if ok {
				name = runtime.FuncForPC(pc).Name()
			}
		}
		elapsed := time.Since(now)
		mutex.Lock()
		defer mutex.Unlock()
		if val, ok := averages[name]; ok == true {
			val.calls++
			val.time += elapsed
			if elapsed > val.max {
				val.max = elapsed
			}
			averages[name] = val
		} else {
			averages[name] = function_measure{
				calls: 1,
				time:  elapsed,
				max:   elapsed,
			}
		}
	}
}

// This prints a table which has the measurement information of all functions.
func Close() {
	mutex.Lock()
	defer mutex.Unlock()

	type funcTimeSummary struct {
		name    string
		calls   int
		time    time.Duration
		average time.Duration
		max     time.Duration
	}

	sorted := []funcTimeSummary{}
	for key, val := range averages {
		sum := funcTimeSummary{
			name:    strings.Replace(key, "main.", "", 1),
			calls:   val.calls,
			time:    val.time,
			average: val.time / time.Duration(val.calls),
			max:     val.max,
		}
		sorted = append(sorted, sum)
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].time > sorted[j].time
	})

	table := tw.NewWriter(os.Stdout)
	table.SetHeader([]string{"NAME", "CALLS", "TIME", "AVERAGE", "MAX"})
	for _, r := range sorted {
		calls := strconv.Itoa(r.calls)
		average := ""
		max := ""
		if r.calls > 1 {
			average = r.average.String()
			max = r.max.String()
		}
		table.Append([]string{r.name, calls, r.time.String(), average, max})
	}
	table.Render()
}
