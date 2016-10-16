package watchdogs

var _ DiffService = &DiffString{}

type DiffString struct {
	b     []byte
	strip int
}

func NewDiffString(diff string, strip int) DiffService {
	return &DiffString{b: []byte(diff), strip: strip}
}

func (d *DiffString) Diff() ([]byte, error) {
	return d.b, nil
}

func (d *DiffString) Strip() int {
	return d.strip
}
