package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
)

// monthMap maps month abbreviations to their numerical values.
var monthMap = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

// numVal represents a parsed numeric value for -n sort.
type numVal struct {
	sign     int
	mantissa float64
	raw      string
}

// humanVal represents a parsed human-numeric value for -h sort.
type humanVal struct {
	sign        int
	suffixOrder int
	mantissa    float64
	raw         string
}

// monthVal represents a parsed month value for -M sort.
type monthVal struct {
	value int
	raw   string
}

// byKey implements sort.Interface for sorting lines based on keys.
type byKey struct {
	lines   []string
	column  int
	numeric bool
	human   bool
	month   bool
	blanks  bool
	reverse bool
}

// Len returns the number of lines.
func (s byKey) Len() int { return len(s.lines) }

// Swap swaps two lines.
func (s byKey) Swap(i, j int) { s.lines[i], s.lines[j] = s.lines[j], s.lines[i] }

// Less compares two lines based on the sort criteria.
func (s byKey) Less(i, j int) bool {
	ki := s.getKey(s.lines[i])
	kj := s.getKey(s.lines[j])
	compare := s.compareKeys(ki, kj)
	less := compare < 0
	if s.reverse {
		less = !less
	}
	return less
}

// getKey extracts the sort key from a line.
func (s byKey) getKey(line string) string {
	if s.column <= 0 {
		return line
	}
	fields := strings.Split(line, "\t")
	if s.column-1 >= len(fields) {
		return ""
	}
	return fields[s.column-1]
}

// compareKeys compares two keys based on the flags.
func (s byKey) compareKeys(a, b string) int {
	keyA := a
	keyB := b
	if s.blanks {
		keyA = strings.TrimRight(keyA, " \t")
		keyB = strings.TrimRight(keyB, " \t")
	}
	trimmedA := strings.TrimLeft(keyA, " \t")
	trimmedB := strings.TrimLeft(keyB, " \t")
	var cmp int
	if s.human {
		ha := parseHuman(trimmedA, keyA)
		hb := parseHuman(trimmedB, keyB)
		cmp = humanCmp(ha, hb)
	} else if s.numeric {
		na := parseNumeric(trimmedA, keyA)
		nb := parseNumeric(trimmedB, keyB)
		cmp = numericCmp(na, nb)
	} else if s.month {
		ma := parseMonth(trimmedA, keyA)
		mb := parseMonth(trimmedB, keyB)
		cmp = monthCmp(ma, mb)
	} else {
		cmp = strings.Compare(keyA, keyB)
	}
	return cmp
}

// parseNumeric parses a string for numeric sort.
func parseNumeric(trimmed, raw string) numVal {
	var hasDigit bool
	i := 0
	neg := false
	if len(trimmed) > 0 {
		if trimmed[0] == '-' {
			neg = true
			i++
		} else if trimmed[0] == '+' {
			i++
		}
	}
	start := i
	hasDot := false
	hasE := false
	for ; i < len(trimmed); i++ {
		c := trimmed[i]
		if c >= '0' && c <= '9' {
			hasDigit = true
		} else if c == '.' && !hasDot && !hasE {
			hasDot = true
		} else if (c == 'e' || c == 'E') && hasDigit && !hasE {
			hasE = true
			hasDot = false
		} else if (c == '+' || c == '-') && hasE && (trimmed[i-1] == 'e' || trimmed[i-1] == 'E') {
			// continue
		} else {
			break
		}
	}
	numStr := trimmed[start:i]
	if !hasDigit {
		numStr = "0"
	}
	v, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		v = 0
	}
	mant := v
	if mant < 0 {
		mant = -mant
		neg = true
	}
	sig := 0
	if mant != 0 {
		if neg {
			sig = -1
		} else {
			sig = 1
		}
	} else {
		if neg {
			sig = -1
		} else {
			sig = 0
		}
	}
	return numVal{sig, mant, raw}
}

// numericCmp compares two numVal.
func numericCmp(na, nb numVal) int {
	cmpSign := cmpInt(na.sign, nb.sign)
	if cmpSign != 0 {
		return cmpSign
	}
	cmpMant := cmpFloat(na.mantissa, nb.mantissa)
	if na.sign == -1 {
		cmpMant = -cmpMant
	}
	if cmpMant != 0 {
		return cmpMant
	}
	return strings.Compare(na.raw, nb.raw)
}

// parseHuman parses a string for human-numeric sort.
func parseHuman(trimmed, raw string) humanVal {
	var hasDigit bool
	i := 0
	neg := false
	if len(trimmed) > 0 {
		if trimmed[0] == '-' {
			neg = true
			i++
		} else if trimmed[0] == '+' {
			i++
		}
	}
	start := i
	hasDot := false
	hasE := false
	for ; i < len(trimmed); i++ {
		c := trimmed[i]
		if c >= '0' && c <= '9' {
			hasDigit = true
		} else if c == '.' && !hasDot && !hasE {
			hasDot = true
		} else if (c == 'e' || c == 'E') && hasDigit && !hasE {
			hasE = true
			hasDot = false
		} else if (c == '+' || c == '-') && hasE && (trimmed[i-1] == 'e' || trimmed[i-1] == 'E') {
			// continue
		} else {
			break
		}
	}
	numStr := trimmed[start:i]
	suffixStr := trimmed[i:]
	if !hasDigit {
		numStr = "0"
	}
	v, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		v = 0
	}
	mant := v
	if mant < 0 {
		mant = -mant
		neg = true
	}
	sig := 0
	if mant != 0 {
		if neg {
			sig = -1
		} else {
			sig = 1
		}
	} else {
		if neg {
			sig = -1
		} else {
			sig = 0
		}
	}
	suffixOrder := 0
	if len(suffixStr) > 0 {
		c := suffixStr[0]
		switch c {
		case 'k', 'K':
			suffixOrder = 1
		case 'M':
			suffixOrder = 2
		case 'G':
			suffixOrder = 3
		case 'T':
			suffixOrder = 4
		case 'P':
			suffixOrder = 5
		case 'E':
			suffixOrder = 6
		case 'Z':
			suffixOrder = 7
		case 'Y':
			suffixOrder = 8
		}
	}
	if !hasDigit {
		suffixOrder = 0
	}
	return humanVal{sig, suffixOrder, mant, raw}
}

// humanCmp compares two humanVal.
func humanCmp(ha, hb humanVal) int {
	cmpSign := cmpInt(ha.sign, hb.sign)
	if cmpSign != 0 {
		return cmpSign
	}
	cmpSuffix := cmpInt(ha.suffixOrder, hb.suffixOrder)
	if cmpSuffix != 0 {
		return cmpSuffix
	}
	cmpMant := cmpFloat(ha.mantissa, hb.mantissa)
	if ha.sign == -1 {
		cmpMant = -cmpMant
	}
	if cmpMant != 0 {
		return cmpMant
	}
	return strings.Compare(ha.raw, hb.raw)
}

// parseMonth parses a string for month sort.
func parseMonth(trimmed, raw string) monthVal {
	if len(trimmed) < 3 {
		return monthVal{0, raw}
	}
	m := strings.ToUpper(trimmed[0:3])
	v, ok := monthMap[m]
	if !ok {
		v = 0
	}
	return monthVal{v, raw}
}

// monthCmp compares two monthVal.
func monthCmp(ma, mb monthVal) int {
	cmpV := cmpInt(ma.value, mb.value)
	if cmpV != 0 {
		return cmpV
	}
	return strings.Compare(ma.raw, mb.raw)
}

// cmpInt compares two integers and returns -1, 0, or 1.
func cmpInt(x, y int) int {
	if x < y {
		return -1
	} else if x > y {
		return 1
	}
	return 0
}

// cmpFloat compares two floats and returns -1, 0, or 1.
func cmpFloat(x, y float64) int {
	if math.IsNaN(x) || math.IsNaN(y) {
		// Handle NaN as greater or something, but for simplicity.
		return strings.Compare(fmt.Sprint(x), fmt.Sprint(y))
	}
	if x < y {
		return -1
	} else if x > y {
		return 1
	}
	return 0
}

func main() {
	column := flag.Int("k", 0, "sort by column N (1-based, default whole line)")
	numeric := flag.Bool("n", false, "sort by numerical value")
	reverse := flag.Bool("r", false, "sort in reverse order")
	unique := flag.Bool("u", false, "output unique lines only")
	month := flag.Bool("M", false, "sort by month name")
	blanks := flag.Bool("b", false, "ignore trailing blanks")
	check := flag.Bool("c", false, "check if data is sorted")
	human := flag.Bool("h", false, "sort by human-readable numeric value")
	flag.Parse()

	if *month && (*numeric || *human) {
		log.Fatal("Cannot combine month and numeric/human sort")
	}
	if *numeric && *human {
		log.Fatal("Cannot combine numeric and human-numeric sort")
	}

	var reader io.Reader
	args := flag.Args()
	if len(args) > 1 {
		log.Fatal("Too many input files; only one file or STDIN supported")
	} else if len(args) == 1 {
		f, err := os.Open(args[0])
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		reader = f
	} else {
		reader = os.Stdin
	}

	lines := []string{}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	sorter := byKey{
		lines:   lines,
		column:  *column,
		numeric: *numeric,
		human:   *human,
		month:   *month,
		blanks:  *blanks,
		reverse: *reverse,
	}

	if *check {
		if !sort.IsSorted(sorter) {
			fmt.Println("Data is not sorted")
			os.Exit(1)
		}
	} else {
		sort.Sort(sorter)
		if *unique {
			uniqLines := []string{}
			for i := 0; i < len(lines); i++ {
				if i == 0 || lines[i] != lines[i-1] {
					uniqLines = append(uniqLines, lines[i])
				}
			}
			lines = uniqLines
		}
		for _, line := range lines {
			fmt.Println(line)
		}
	}
}
