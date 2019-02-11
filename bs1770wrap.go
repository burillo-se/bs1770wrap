package bs1770wrap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"os/exec"
	"regexp"
	"strconv"
)

// LoudnessData struct used to return result of
// running bs1770gain and calculating gain, as
// well as running sox and calculating length
type LoudnessData struct {
	Integrated float32 // lufs
	Peak       float32 // lufs
	Range      float32 // lufs
	Shortterm  float32 // lufs
	Momentary  float32 // lufs
	Length     int64   // microseconds
}

/* Data format:

`
<bs1770gain>
  <album>
    <track total="29" number="1" file="01&#x20;&#x2D;&#x20;Powerful&#x20;Blues&#x20;Rock&#x2E;wav">
      <integrated lufs="-14.14" lu="-8.86" />
      <momentary lufs="-9.55" lu="-13.45" />
      <shortterm-maximum lufs="-11.32" lu="-11.68" />
      <range lufs="4.52" />
      <true-peak tpfs="0.05" factor="1.005459" />
    </track>
  </album>
</bs1770gain>
`

We ignore the summary part, as well as ignore everything else.
*/

type integratedData struct {
	XMLName xml.Name `xml:"integrated"`
	Value   float32  `xml:"lufs,attr"`
}

type rangeData struct {
	XMLName xml.Name `xml:"range"`
	Value   float32  `xml:"lufs,attr"`
}

type truePeakData struct {
	XMLName xml.Name `xml:"true-peak"`
	Value   float32  `xml:"tpfs,attr"`
}

type momentaryMaximumData struct {
	XMLName xml.Name `xml:"momentary"`
	Value float32    `xml:"lufs,attr"`
}

type shorttermMaximumData struct {
	XMLName xml.Name `xml:"shortterm-maximum"`
	Value float32    `xml:"lufs,attr"`
}

type trackData struct {
	XMLName            xml.Name `xml:"track"`
	Integrated         integratedData
	MomentaryMaximum   momentaryMaximumData
	ShorttermMaximum   shorttermMaximumData
	Range              rangeData
	TruePeak           truePeakData
}

type albumData struct {
	XMLName xml.Name `xml:"album"`
	Track   trackData
}

type bs1770gainData struct {
	XMLName xml.Name `xml:"bs1770gain"`
	Album   albumData
}

// CalculateLoudness will take in a path to an audio file,
// analyze it with bs1770gain, and return a struct populated
// with data we're interested in. To avoid bass-heavy music
// skewing the measurements, we'll be using sox to highpass
// the file before scanning it for loudness.
func CalculateLoudness(file string) (LoudnessData, error) {
	var out bytes.Buffer

	sampleRegex, err := regexp.Compile(`Length \(seconds\):\s+(?P<len>\d+(\.\d+)?)`)
	if err != nil {
		return LoudnessData{}, fmt.Errorf("Cannot compile regex: %v", err)
	}

	// write a hi-passed file into temporary dir
	cmd := exec.Command("sox",
		file,
		"-n",
		"stat",
	)
	cmd.Stderr = &out

	err = cmd.Run()
	if err != nil {
		return LoudnessData{}, fmt.Errorf("Error creating temporary file: %v", err)
	}

	// get length from regex
	matches := sampleRegex.FindStringSubmatch(out.String())

	result := make(map[string]string)
	for i, name := range matches {
		result[sampleRegex.SubexpNames()[i]] = name
	}
	lenstr, ok := result["len"]
	if !ok {
		return LoudnessData{}, fmt.Errorf("Cannot get audio length: regex did not match")
	}

	len64, err := strconv.ParseFloat(lenstr, 32)
	if err != nil {
		return LoudnessData{}, fmt.Errorf("Cannot parse audio length: %v", err)
	}
	out.Reset()

	cmd = exec.Command("bs1770gain",
		"-itrms",           // integrated, true peak, range, momentary, shortterm
		"--loglevel=quiet", // remove all non-essential output
		"--xml",            // get XML output
		file,            // what file to scan
	)

	cmd.Stdout = &out

	err = cmd.Run()
	if err != nil {
		return LoudnessData{}, fmt.Errorf("Cannot calculate loudness: %v", err)
	}

	gd := bs1770gainData{}
	err = xml.Unmarshal([]byte(out.String()), &gd)
	if err != nil {
		return LoudnessData{}, fmt.Errorf("Cannot parse loudness information: %v", err)
	}

	microseconds := int64(math.Round(len64 * 1000000.0))

	return LoudnessData{
		Integrated: gd.Album.Track.Integrated.Value,
		Range:      gd.Album.Track.Range.Value,
		Peak:       gd.Album.Track.TruePeak.Value,
		Shortterm:  gd.Album.Track.ShorttermMaximum.Value,
		Momentary:  gd.Album.Track.MomentaryMaximum.Value,
		Length:     microseconds,
	}, nil
}
