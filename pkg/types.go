package pkg

import "fmt"

type BadStatusError struct {
	URL    string
	Status int
}

func (b BadStatusError) Error() string {
	return fmt.Sprintf("did not get a successful response from %s, got %d", b.URL, b.Status)
}
