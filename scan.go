package router

import "strings"

// pathScanner implements a path element scanner.
type pathScanner struct {
	path string
}

// Resets the scanner state to read from path.
func (s *pathScanner) Reset(path string) {
	s.path = strings.TrimPrefix(path, "/")
}

// Len returns the remaining length of the Path.
func (s *pathScanner) Len() int {
	return len(s.path)
}

// Next returns the next element in the path or nil.
func (s *pathScanner) Next() string {
	if s.Len() == 0 {
		return ""
	}
	i := strings.Index(s.path, "/")
	if i < 0 {
		str := s.path
		s.path = ""
		return str
	}
	str := s.path[:i]
	s.path = s.path[i+1:]
	return str
}
