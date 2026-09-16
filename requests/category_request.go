package requests

import "fmt"

type CategoryRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Image       string `json:"image"`
}

func (r *CategoryRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("nama kategori wajib diisi")
	}
	if len(r.Name) > 100 {
		return fmt.Errorf("nama kategori maksimal 100 karakter")
	}
	return nil
}

