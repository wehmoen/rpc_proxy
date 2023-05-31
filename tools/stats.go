package tools

type Request map[string]int

func (r Request) Add(method string) {
	r[method] += 1
}
