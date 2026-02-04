package model

type Track struct {
	Path     string
	Title    string
	Artist   string
	Album    string
	Genre    string
	Year     int
	TrackNum int
	Duration float64
	Mtime    int64
	Checksum string
	Key      string
	BPM      float64
	Favorite bool
}
