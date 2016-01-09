package main

type StringPool map[string]*string

var STRINGS StringPool = StringPool{}

func (p StringPool) Intern(s string) *string {
	ss, exists := p[s]
	if exists {
		return ss
	}
	p[s] = &s
	return &s
}
