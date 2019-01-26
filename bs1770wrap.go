package bs1770wrap

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os/exec"
)

// LoudnessData struct used to return result of
// running bs1770 and calculating gain
type LoudnessData struct {
	IntegratedLoudness float32
	TruePeak           float32
	LoudnessRange      float32
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
	cmd := exec.Command("bs1770gain",
		"-itr",             // integrated, true peak, range
		"--loglevel=quiet", // remove all non-essential output
		"--xml",            // get XML output
		file,               // what file to scan
	)

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
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

	return ld, nil
}
