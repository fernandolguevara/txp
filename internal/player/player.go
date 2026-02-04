package player

type Player interface {
	Load(path string) error
	TestTone(seconds float64) error
	TogglePause() error
	AddVolume(delta int) error
	SetVolume(value int) error
	GetVolume() int
	Seek(seconds float64) error
	GetStringProperty(name string) (string, bool, error)
	GetFloatProperty(name string) (float64, bool, error)
	Close()
}

type EOFNotifier interface {
	SetOnEOF(func())
}
