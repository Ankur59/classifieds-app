package models

// Student mirrors a row in the students table.
type Student struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
