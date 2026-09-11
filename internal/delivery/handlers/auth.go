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
			w.Header().Set("HX-Redirect", "/login?error=inactive_user")
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("HX-Redirect", "/login?error=invalid_credentials")
		w.WriteHeader(http.StatusOK)
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

	h.renderer.Render(w, "register", usuarioFormData{
		Title:    "Registrar Usuario",
		AppName:  appName,
		Error:    r.URL.Query().Get("error"),
		Success:  r.URL.Query().Get("success"),
		Usuario:  &identity.Usuario{},
		IsAdmin:  session.Nivel >= 4,
		Session:  session,
		EditMode: false,
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
		h.renderRegisterError(w, session, "empty_fields")
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
		h.renderRegisterError(w, session, "empty_fields")
		return
	}

	if password != confirmPassword {
		h.renderRegisterError(w, session, "password_mismatch")
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
			h.renderRegisterError(w, session, "user_exists")
			return
		}
		if errors.Is(err, identity.ErrEmailInvalido) {
			h.renderRegisterError(w, session, "invalid_email")
			return
		}
		h.renderRegisterError(w, session, "registration_failed")
		return
	}

	w.Header().Set("HX-Redirect", "/usuarios?success=usuario_actualizado")
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) renderRegisterError(w http.ResponseWriter, session *middleware.SessionData, errKey string) {
	data := usuarioFormData{
		Title:    "Registrar Usuario",
		AppName:  appName,
		Session:  session,
		Usuario:  &identity.Usuario{},
		Error:    errKey,
		EditMode: false,
		IsAdmin:  session.Nivel >= 4,
	}
	h.renderer.RenderPartial(w, "form_usuario", data, http.StatusOK)
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
