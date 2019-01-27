package bs1770wrap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
)

// LoudnessData struct used to return result of
// running bs1770gain and calculating gain, as
// well as running sox and calculating length
type LoudnessData struct {
	IntegratedLoudness float32 // lufs
	TruePeak           float32 // lufs
	LoudnessRange      float32 // lufs
	Length             float32 // seconds
}

/* Data format:

`<bs1770gain>
  <album>
    <track total="1" number="1" file="audio&#x2E;mp3">
      <integrated lufs="-16.32" lu="-6.68" />
      <range lufs="2.53" />
      <true-peak tpfs="-0.23" factor="0.974183" />
    </track>
    <summary total="1">
      <integrated lufs="-16.32" lu="-6.68" />
      <range lufs="2.53" />
      <true-peak tpfs="-0.23" factor="0.974183" />
    </summary>
  </album>
</bs1770gain>
`

We ignore the summary part, as well as ignore everything else.
*/

type integratedLoudnessData struct {
	XMLName xml.Name `xml:"integrated"`
	Value   float32  `xml:"lufs,attr"`
}

type loudnessRangeData struct {
	XMLName xml.Name `xml:"range"`
	Value   float32  `xml:"lufs,attr"`
}

type truePeakData struct {
	XMLName xml.Name `xml:"true-peak"`
	Value   float32  `xml:"tpfs,attr"`
}

type trackData struct {
	XMLName            xml.Name `xml:"track"`
	IntegratedLoudness integratedLoudnessData
	LoudnessRange      loudnessRangeData
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
// with data we're interested in.
func CalculateLoudness(file string) (LoudnessData, error) {
	var out bytes.Buffer

	sampleRegex, err := regexp.Compile(`Length \(seconds\):\s+(?P<len>\d+(\.\d+)?)`)
	if err != nil {
		return LoudnessData{}, fmt.Errorf("Cannot compile regex: %v", err)
	}

	cmd := exec.Command("sox",
		file, // file to scan
		"-n", // gather statistics
		"stat",
	)
	cmd.Stderr = &out

	err = cmd.Run()
	if err != nil {
		return LoudnessData{}, fmt.Errorf("Cannot get audio length: %v", err)
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
		"-itr",             // integrated, true peak, range
		"--loglevel=quiet", // remove all non-essential output
		"--xml",            // get XML output
		file,               // what file to scan
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

	ld := LoudnessData{}
	ld.IntegratedLoudness = gd.Album.Track.IntegratedLoudness.Value
	ld.LoudnessRange = gd.Album.Track.LoudnessRange.Value
	ld.TruePeak = gd.Album.Track.TruePeak.Value
	ld.Length = float32(len64)

	return ld, nil
}
