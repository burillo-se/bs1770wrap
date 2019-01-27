# bs1770wrap
A simple Go wrapper around running bs1770gain.

This only extracts a very limited set of information.

Needs:
- sox (length detection)
- libsox-fmt-mp3 (MP3 format support for sox)
- bs1770gain (loudness detection) [^1]

[^1] depending on the distro, bs1770gain version in your repo may be buggy, so it is recommended either to compile it from source, or use precompiled binaries from the project webpage: https://sourceforge.net/projects/bs1770gain/

Using, creating or contributing to this package is in no way intended as an endorsement of bs1770gain author's political views.
