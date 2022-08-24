package customaccesslog

import "testing"

func Test_Ignores(t *testing.T) {

	dataMatchs := map[string][]string{
		"^/aaa": []string{
			"/aaa",
			"/aaabbb",
			"/aaa/xxx",
		},
		"^/xxx/[a-z0-9]+$": []string{
			"/xxx/1",
			"/xxx/aaa",
		},
	}

	for pattern, strs := range dataMatchs {
		Ignore(pattern)
		for _, s := range strs {
			if !Match(s) {
				t.Errorf("pattern [%s] match [%s] fail\n", pattern, s)
			}
		}
	}
}
