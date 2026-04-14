package httpapi

import (
	"net/http"
	"strings"

	"taskflow-backend/internal/validator"
)

type registerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  *domainUser `json:"user"`
}

type domainUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func handleAuthRegister(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Auth == nil {
			return ServiceUnavailable("auth not configured")
		}

		var req registerRequest
		if err := decodeJSON(r, &req); err != nil {
			return BadJSON()
		}

		req.Name = strings.TrimSpace(req.Name)
		req.Email = strings.TrimSpace(req.Email)
		v := validator.New()
		v.Required("name", req.Name)
		v.Required("email", req.Email)
		v.Required("password", req.Password)
		if !v.Ok() {
			return Validation(v.Fields)
		}

		tok, u, err := d.Auth.Register(r.Context(), req.Name, req.Email, req.Password)
		if err != nil {
			return err
		}

		writeJSON(w, http.StatusCreated, authResponse{
			Token: tok,
			User:  &domainUser{ID: u.ID, Name: u.Name, Email: u.Email},
		})
		return nil
	}
}

func handleAuthLogin(d Deps) handlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		if d.Auth == nil {
			return ServiceUnavailable("auth not configured")
		}

		var req loginRequest
		if err := decodeJSON(r, &req); err != nil {
			return BadJSON()
		}
		req.Email = strings.TrimSpace(req.Email)
		v := validator.New()
		v.Required("email", req.Email)
		v.Required("password", req.Password)
		if !v.Ok() {
			return Validation(v.Fields)
		}

		tok, u, err := d.Auth.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			return err
		}

		writeJSON(w, http.StatusOK, authResponse{
			Token: tok,
			User:  &domainUser{ID: u.ID, Name: u.Name, Email: u.Email},
		})
		return nil
	}
}
