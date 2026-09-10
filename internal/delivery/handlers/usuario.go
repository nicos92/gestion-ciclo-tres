package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"gestion-ciclo-tres/internal/delivery/middleware"
	"gestion-ciclo-tres/internal/delivery/render"
	"gestion-ciclo-tres/internal/identity"
	"gestion-ciclo-tres/internal/tarima"
)

type UsuarioHandler struct {
	renderer *render.Renderer
	auth     *identity.AuthService
	tarima   *tarima.TarimaService
	sessions *middleware.SessionStore
}

func NewUsuarioHandler(r *render.Renderer, a *identity.AuthService, ts *tarima.TarimaService, s *middleware.SessionStore) *UsuarioHandler {
	return &UsuarioHandler{renderer: r, auth: a, tarima: ts, sessions: s}
}

type dashboardPageData struct {
	Title        string
	AppName      string
	Session      *middleware.SessionData
	TotalTarimas int
	TotalUsuarios int
}

type usuariosPageData struct {
	Title   string
	AppName string
	Session *middleware.SessionData
	Usuarios []identity.Usuario
	Success string
}

type editarUsuarioPageData struct {
	Title   string
	AppName string
	Session *middleware.SessionData
	Usuario *identity.Usuario
	Error   string
	Success string
}

func (h *UsuarioHandler) ShowDashboard(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)

	totalTarimas, err := h.tarima.CountAll(r.Context())
	if err != nil {
		slog.Error("contar tarimas para dashboard", "error", err)
		totalTarimas = 0
	}

	totalUsuarios := 0
	if session.Nivel >= 4 {
		totalUsuarios, err = h.auth.CountAll(r.Context())
		if err != nil {
			slog.Error("contar usuarios para dashboard", "error", err)
			totalUsuarios = 0
		}
	}

	data := dashboardPageData{
		Title:         "Panel de Control",
		AppName:       appName,
		Session:       session,
		TotalTarimas:  totalTarimas,
		TotalUsuarios: totalUsuarios,
	}
	h.renderer.Render(w, "dashboard", data, http.StatusOK)
}

func (h *UsuarioHandler) ListarUsuarios(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)

	usuarios, err := h.auth.ListAll(r.Context())
	if err != nil {
		slog.Error("listar usuarios", "error", err)
		http.Error(w, "Error interno al listar usuarios", http.StatusInternalServerError)
		return
	}

	data := usuariosPageData{
		Title:    "Gestión de Usuarios",
		AppName:  appName,
		Session:  session,
		Usuarios: usuarios,
		Success:  r.URL.Query().Get("success"),
	}
	h.renderer.Render(w, "usuarios", data, http.StatusOK)
}

func (h *UsuarioHandler) ShowEditarUsuario(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/usuarios", http.StatusFound)
		return
	}

	u, err := h.auth.GetByID(r.Context(), id)
	if err != nil {
		http.Redirect(w, r, "/usuarios", http.StatusFound)
		return
	}

	data := editarUsuarioPageData{
		Title:   "Editar Usuario",
		AppName: appName,
		Session: session,
		Usuario: u,
		Error:   r.URL.Query().Get("error"),
		Success: r.URL.Query().Get("success"),
	}
	h.renderer.Render(w, "editar_usuario", data, http.StatusOK)
}

func (h *UsuarioHandler) ActualizarUsuario(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/usuarios", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/usuarios/editar/"+strconv.FormatInt(id, 10)+"?error=update_failed", http.StatusFound)
		return
	}

	firstName := strings.TrimSpace(r.FormValue("firstName"))
	lastName := strings.TrimSpace(r.FormValue("lastName"))
	email := strings.TrimSpace(r.FormValue("email"))
	username := strings.TrimSpace(r.FormValue("username"))
	legajo := strings.TrimSpace(r.FormValue("legajo"))
	department := r.FormValue("department")
	newPassword := r.FormValue("newPassword")
	confirmPassword := r.FormValue("confirmPassword")
	activo := r.FormValue("activo") == "on"

	if firstName == "" || lastName == "" || email == "" || username == "" || legajo == "" {
		http.Redirect(w, r, "/usuarios/editar/"+strconv.FormatInt(id, 10)+"?error=empty_fields", http.StatusFound)
		return
	}

	if newPassword != "" || confirmPassword != "" {
		if newPassword == "" || confirmPassword == "" {
			http.Redirect(w, r, "/usuarios/editar/"+strconv.FormatInt(id, 10)+"?error=empty_password_fields", http.StatusFound)
			return
		}
		if newPassword != confirmPassword {
			http.Redirect(w, r, "/usuarios/editar/"+strconv.FormatInt(id, 10)+"?error=password_mismatch", http.StatusFound)
			return
		}
		if len(newPassword) < 6 {
			http.Redirect(w, r, "/usuarios/editar/"+strconv.FormatInt(id, 10)+"?error=weak_password", http.StatusFound)
			return
		}
	}

	idRol := 4
	if v := r.FormValue("idRol"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			idRol = n
		}
	}

	u := &identity.Usuario{
		ID:         id,
		Username:   username,
		Email:      email,
		FirstName:  firstName,
		LastName:   lastName,
		Legajo:     legajo,
		Department: department,
		Rol:        identity.Rol{ID: int64(idRol)},
		Activo:     activo,
	}

	if err := h.auth.Update(r.Context(), u, newPassword); err != nil {
		slog.Warn("actualización de usuario fallida", "error", err)
		errKey := "update_failed"
		if err == identity.ErrUsernameExiste || err == identity.ErrEmailExiste {
			errKey = "user_exists"
		} else if err == identity.ErrEmailInvalido {
			errKey = "invalid_email"
		} else if err == identity.ErrCamposRequeridos {
			errKey = "empty_fields"
		}
		http.Redirect(w, r, "/usuarios/editar/"+strconv.FormatInt(id, 10)+"?error="+errKey, http.StatusFound)
		return
	}

	http.Redirect(w, r, "/usuarios?success=usuario_actualizado", http.StatusFound)
}
