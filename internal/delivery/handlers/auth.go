package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"gestion-ciclo-tres/internal/delivery/middleware"
	"gestion-ciclo-tres/internal/delivery/render"
	"gestion-ciclo-tres/internal/identity"
)

type AuthHandler struct {
	renderer *render.Renderer
	auth     *identity.AuthService
	sessions *middleware.SessionStore
}

func NewAuthHandler(r *render.Renderer, a *identity.AuthService, s *middleware.SessionStore) *AuthHandler {
	return &AuthHandler{renderer: r, auth: a, sessions: s}
}

type loginPageData struct {
	Title   string
	AppName string
	Error   string
	Session *middleware.SessionData
}

func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)
	if session != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	h.renderer.Render(w, "login", loginPageData{
		Title:   "Iniciar Sesión",
		AppName: "Gestión de Tarimas",
		Error:   r.URL.Query().Get("error"),
	}, http.StatusOK)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/login?error=empty_fields", http.StatusFound)
		return
	}

	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")

	u, err := h.auth.Login(r.Context(), username, password)
	if err != nil {
		if errors.Is(err, identity.ErrUsuarioInactivo) {
			http.Redirect(w, r, "/login?error=inactive_user", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/login?error=invalid_credentials", http.StatusFound)
		return
	}

	sessionID := h.sessions.Create(&middleware.SessionData{
		UserID:    u.ID,
		Username:  u.Username,
		Nivel:     u.Rol.Nivel,
		NombreRol: u.Rol.NombreRol,
	})
	middleware.SetSessionCookie(w, sessionID)

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) ShowRegister(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	h.renderer.Render(w, "register", struct {
		Title     string
		AppName   string
		Error     string
		Success   bool
		IsAdmin   bool
		Session   *middleware.SessionData
	}{
		Title:   "Registrar Usuario",
		AppName: "Gestión de Tarimas",
		Error:   r.URL.Query().Get("error"),
		Success: r.URL.Query().Get("success") == "true",
		IsAdmin: session.Nivel >= 4,
		Session: session,
	}, http.StatusOK)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)
	if session == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}
	if session.Nivel < 4 {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/register?error=empty_fields", http.StatusFound)
		return
	}

	firstName := strings.TrimSpace(r.FormValue("firstName"))
	lastName := strings.TrimSpace(r.FormValue("lastName"))
	email := strings.TrimSpace(r.FormValue("email"))
	username := strings.TrimSpace(r.FormValue("username"))
	legajo := strings.TrimSpace(r.FormValue("legajo"))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirmPassword")
	department := r.FormValue("department")

	if firstName == "" || lastName == "" || email == "" || username == "" || legajo == "" || password == "" || confirmPassword == "" || department == "" {
		http.Redirect(w, r, "/register?error=empty_fields", http.StatusFound)
		return
	}

	if password != confirmPassword {
		http.Redirect(w, r, "/register?error=password_mismatch", http.StatusFound)
		return
	}

	idRol := 4
	if v := r.FormValue("idRol"); v != "" {
		switch v {
		case "1":
			idRol = 1
		case "2":
			idRol = 2
		case "3":
			idRol = 3
		default:
			idRol = 4
		}
	}

	activo := r.FormValue("activo") == "on"

	u := &identity.Usuario{
		Username:   username,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		Legajo:     legajo,
		Department: department,
		Rol:        identity.Rol{ID: int64(idRol)},
		Activo:     activo,
	}

	_, err := h.auth.Register(r.Context(), u, password)
	if err != nil {
		slog.Warn("registro fallido", "error", err)
		if errors.Is(err, identity.ErrUsernameExiste) || errors.Is(err, identity.ErrEmailExiste) {
			http.Redirect(w, r, "/register?error=user_exists", http.StatusFound)
			return
		}
		if errors.Is(err, identity.ErrEmailInvalido) {
			http.Redirect(w, r, "/register?error=invalid_email", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/register?error=registration_failed", http.StatusFound)
		return
	}

	http.Redirect(w, r, "/register?success=true", http.StatusFound)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	sid := middleware.GetSessionID(r)
	if sid != "" {
		h.sessions.Delete(sid)
	}
	middleware.ClearSessionCookie(w)

	w.Header().Set("HX-Redirect", "/login")
	w.WriteHeader(http.StatusOK)
}
