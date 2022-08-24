package customaccesslog

import "regexp"

type Ignores struct {
	ps []*regexp.Regexp
}

var defaultIgnores = &Ignores{
	ps: []*regexp.Regexp{},
}

func Ignore(pattern string) {
	defaultIgnores.ps = append(defaultIgnores.ps, regexp.MustCompile(pattern))
}

func Match(str string) bool {
	b := []byte(str)
	for _, re := range defaultIgnores.ps {
		if re.Match(b) {
			return true
		}
	}
	return false
}
