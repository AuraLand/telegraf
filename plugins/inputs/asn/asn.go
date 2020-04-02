package asn

import (
	"bufio"
	"fmt"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/filter"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/inputs/system"
	"os"
	"strconv"
	"strings"
)

type ASNStats struct {
	filter filter.Filter
	ps     system.PS

	Interfaces []string
}

func (_ *ASNStats) Description() string {
	return "Read metric about asn usage"
}

var sampleConfig = `
  ## Need to be done
  # interfaces = ["eth0"]
`

func (_ *ASNStats) SampleConfig() string {
	return sampleConfig
}

func (s *ASNStats) Gather(acc telegraf.Accumulator) error {
	tags := map[string]string{
		"interface": "ens5",
	}
	var pkts_received uint64
	lines, err := ReadLines("/var/run/asn-ddos/stats/received")
	if err != nil {
		//return fmt.Errorf("error getting asn pkts_received info: %s", err)
		pkts_received = 0
	} else {
		if len(lines) != 1 {
			return fmt.Errorf("wrong pkts_received format")
		}
		fields := strings.Fields(lines[0])
		received, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return fmt.Errorf("%s", err)
		}
		pkts_received = received

	}

	var pkts_dropped uint64
	lines, err = ReadLines("/var/run/asn-ddos/stats/dropped")
	if err != nil {
		//return fmt.Errorf("error getting asn pkts_received info: %s", err)
		pkts_dropped = 0
	} else {
		if len(lines) != 1 {
			return fmt.Errorf("wrong pkts_received format")
		}
		fields := strings.Fields(lines[0])
		dropped, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return fmt.Errorf("%s", err)
		}
		pkts_dropped = dropped
	}

	fields := map[string]interface{}{
		"pkts_received": pkts_received,
		"pkts_dropped":  pkts_dropped,
	}

	acc.AddCounter("asn", fields, tags)
	return nil
}
func init() {
	inputs.Add("asn", func() telegraf.Input {
		return &ASNStats{ps: system.NewSystemPS()}
	})
}

// ReadLines reads contents from a file and splits them by new lines.
// A convenience wrapper to ReadLinesOffsetN(filename, 0, -1).
func ReadLines(filename string) ([]string, error) {
	return ReadLinesOffsetN(filename, 0, -1)
}

// ReadLines reads contents from file and splits them by new line.
// The offset tells at which line number to start.
// The count determines the number of lines to read (starting from offset):
//   n >= 0: at most n lines
//   n < 0: whole file
func ReadLinesOffsetN(filename string, offset uint, n int) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string

	r := bufio.NewReader(f)
	for i := 0; i < n+int(offset) || n < 0; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		if i < int(offset) {
			continue
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}

	return ret, nil
}
