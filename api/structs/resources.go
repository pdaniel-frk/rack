package structs

import "time"

type Resource struct {
	Id   string
	Name string

	Reason string
	Status string
	Type   string

	Time time.Time
}

type Resources map[string]Resource
