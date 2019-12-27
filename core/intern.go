package core

import (
	"fmt"
	"strconv"
)

type (
	StringPool map[string]*string
)

func (p StringPool) Intern(s string) *string {
	ss, exists := p[s]
	if exists {
		return ss
	}
	p[s] = &s
	return &s
}

func (p StringPool) InternExistingString(s *string) {
	ss, exists := p[*s]
	if exists {
		if ss != s {
			panic(fmt.Sprintf("New string %s does not match existing string %s", strconv.Quote(*s), strconv.Quote(*ss)))
		}
		return
	}
	p[*s] = s
}
