package types

type AlarmArgs struct {
	Name     string `json:"Name,omitempty"`
	Product  string `json:"Product,omitempty"`
	Priority string `json:"Priority,omitempty"`
	Content  string `json:"Content,omitempty"`
	NoDeal   string `json:"NoDeal,omitempty"`
}
