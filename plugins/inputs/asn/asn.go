package asn

import (
	"bufio"
	"fmt"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/filter"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/inputs/system"
	"net"
	"os"
	"strconv"
	"strings"
)

type ASNStats struct {
	filter filter.Filter
	ps     system.PS

	Interfaces []string
	Virtual    bool
}

func (_ *ASNStats) Description() string {
	return "Read metric about asn usage"
}

var sampleConfig = `
  ## Need to be done
  # interfaces = ["eth0"]
  # virtual = true
`

func (_ *ASNStats) SampleConfig() string {
	return sampleConfig
}

func (s *ASNStats) Gather(acc telegraf.Accumulator) error {
	var factor uint64 = 1
	if !s.Virtual {
		factor = 1024
	}
	netio, err := s.ps.NetIO()
	if err != nil {
		return fmt.Errorf("error getting net io info: %s", err)
	}
	if s.filter == nil {
		if s.filter, err = filter.Compile(s.Interfaces); err != nil {
			return fmt.Errorf("error compiling filter: %s", err)
		}
	}
	interfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("error getting list of interfaces: %s", err)
	}
	interfacesByName := map[string]net.Interface{}
	for _, iface := range interfaces {
		interfacesByName[iface.Name] = iface
	}

	for _, io := range netio {
		if len(s.Interfaces) != 0 {
			var found bool

			if s.filter.Match(io.Name) {
				found = true
			}

			if !found {
				continue
			}
		}

		tags := map[string]string{
			"interface": io.Name,
		}

		/*
			Above is the code copied from ./net/net.go, it is for filter the interface listed in .config file
			Below is customized code for reading data from the file we set in service node
		*/
		pktsReceived, err := readNumberFromFile("/var/run/asn/dms/" + io.Name + "/received")
		if err != nil {
			pktsReceived = 0
		}
		pktsDropped, err := readNumberFromFile("/var/run/asn/dms/" + io.Name + "/dropped")
		if err != nil {
			pktsDropped = 0
		}
		fields := map[string]interface{}{
			"pkts_received": pktsReceived * factor,
			"pkts_dropped":  pktsDropped * factor,
			"bytes_sent":    io.BytesSent * factor,
			"bytes_recv":    io.BytesRecv * factor,
			"packets_sent":  io.PacketsSent * factor,
			"packets_recv":  io.PacketsRecv * factor,
			"err_in":        io.Errin * factor,
			"err_out":       io.Errout * factor,
			"drop_in":       io.Dropin * factor,
			"drop_out":      io.Dropout * factor,
		}
		acc.AddCounter("asn", fields, tags)
	}
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

func readNumberFromFile(filePath string) (uint64, error) {
	var result uint64
	lines, err := ReadLines(filePath)
	if err != nil {
		return 0, fmt.Errorf("error getting asn data info: %s", err)
	} else {
		if len(lines) != 1 {

			return 0, fmt.Errorf("wrong result format")
		}
		fields := strings.Fields(lines[0])
		data, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			fmt.Errorf("%s", err)
		}
		result = data
	}
	return result, nil
}
