package requests

import (
	"fmt"
	"regexp"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Phone    string `json:"phone"`
}

func (r *RegisterRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("nama pengguna wajib diisi")
	}
	if len(r.Name) > 100 {
		return fmt.Errorf("nama pengguna maksimal 100 karakter")
	}
	if r.Email == "" {
		return fmt.Errorf("email wajib diisi")
	}
	if !emailRegex.MatchString(r.Email) {
		return fmt.Errorf("format email tidak valid")
	}
	if r.Password == "" {
		return fmt.Errorf("password wajib diisi")
	}
	if len(r.Password) < 6 {
		return fmt.Errorf("password minimal 6 karakter")
	}
	if r.Phone == "" {
		return fmt.Errorf("telepon wajib diisi")
	}
	if len(r.Phone) > 20 {
		return fmt.Errorf("telepon maksimal 20 karakter")
	}

	return nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return fmt.Errorf("email wajib diisi")
	}

	if r.Password == "" {
		return fmt.Errorf("password wajib diisi")
	}

	if len(r.Password) < 6 {
		return fmt.Errorf("password minimal 6 karakter")
	}

	return nil
}

type UpdateProfileRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
	Profile  string `json:"profile"`
}

func (r *UpdateProfileRequest) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("nama pengguna wajib diisi")
	}
	if len(r.Name) > 100 {
		return fmt.Errorf("nama pengguna maksimal 100 karakter")
	}
	if r.Email == "" {
		return fmt.Errorf("email wajib diisi")
	}
	if !emailRegex.MatchString(r.Email) {
		return fmt.Errorf("format email tidak valid")
	}
	if r.Phone == "" {
		return fmt.Errorf("telepon wajib diisi")
	}
	if len(r.Phone) > 20 {
		return fmt.Errorf("telepon maksimal 20 karakter")
	}
	// Password opsional, tapi kalau diisi minimal 6 karakter
	if r.Password != "" && len(r.Password) < 6 {
		return fmt.Errorf("password minimal 6 karakter")
	}

	return nil
}