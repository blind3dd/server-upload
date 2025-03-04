package writer

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
)

var (
	LineSeparator     = "\r\n"
	entryCount    int = -1
)

type Journal struct {
	entries []string
}

func (j *Journal) Stringer() string {
	return strings.Join(j.entries, "\n")
}

func (j *Journal) AddEntry(text string) int {
	entryCount++
	entry := fmt.Sprintf("%d: %s",
		entryCount, text,
	)
	j.entries = append(j.entries, entry)
	return entryCount
}

func (j *Journal) Save(filename string) {
	perms := fs.FileMode(int64(0644))
	err := os.WriteFile(filename, []byte(j.Stringer()), perms)
	if err != nil {
		panic(err)
	}
}

type Persistence struct {
	LineSeparator string
}

func (p *Persistence) SaveToFile(j *Journal, filename string) {
	perms := fs.FileMode(int(0644))
	err := os.WriteFile(filename,
		[]byte(strings.Join(j.entries, p.LineSeparator)),
		perms,
	)
	if err != nil {
		panic(err)
	}
}

func SaveToFile(j *Journal, filename string) {
	perms := fs.FileMode(int(0644))
	err := os.WriteFile(filename,
		[]byte(strings.Join(j.entries, LineSeparator)),
		perms,
	)
	if err != nil {
		panic(err)
	}
}

func NewJournal() *Journal {
	return &Journal{}
}

// func (j *Journal) Load(filename string) {
// 	panic("implement me")
// }

// func (j *Journal) LoadFromWeb(url *url.URL) {
// 	panic("implement me")
// }
