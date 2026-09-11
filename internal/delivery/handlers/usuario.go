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
	Title         string
	AppName       string
	Session       *middleware.SessionData
	TotalTarimas  int
	TotalUsuarios int
}

type usuariosPageData struct {
	Title    string
	AppName  string
	Session  *middleware.SessionData
	Usuarios []identity.Usuario
	Success  string
}

type usuarioFormData struct {
	Title    string
	AppName  string
	Session  *middleware.SessionData
	Usuario  *identity.Usuario
	Error    string
	Success  string
	EditMode bool
	IsAdmin  bool
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

	data := usuarioFormData{
		Title:    "Editar Usuario",
		AppName:  appName,
		Session:  session,
		Usuario:  u,
		Error:    r.URL.Query().Get("error"),
		Success:  r.URL.Query().Get("success"),
		EditMode: true,
		IsAdmin:  true,
	}
	h.renderer.Render(w, "editar_usuario", data, http.StatusOK)
}

func (h *UsuarioHandler) ActualizarUsuario(w http.ResponseWriter, r *http.Request) {
	session := middleware.SessionFromContext(r)

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Redirect(w, r, "/usuarios", http.StatusFound)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.renderUsuarioFormError(w, session, "", id, "update_failed")
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
		h.renderUsuarioFormError(w, session, "empty_fields", id, "")
		return
	}

	if newPassword != "" || confirmPassword != "" {
		if newPassword == "" || confirmPassword == "" {
			h.renderUsuarioFormError(w, session, "empty_password_fields", id, "")
			return
		}
		if newPassword != confirmPassword {
			h.renderUsuarioFormError(w, session, "password_mismatch", id, "")
			return
		}
		if len(newPassword) < 6 {
			h.renderUsuarioFormError(w, session, "weak_password", id, "")
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
		switch err {
		case identity.ErrUsernameExiste, identity.ErrEmailExiste:
			errKey = "user_exists"
		case identity.ErrEmailInvalido:
			errKey = "invalid_email"
		case identity.ErrCamposRequeridos:
			errKey = "empty_fields"
		}
		h.renderUsuarioFormError(w, session, errKey, id, "")
		return
	}

	updated, _ := h.auth.GetByID(r.Context(), id)
	data := usuarioFormData{
		Title:    "Editar Usuario",
		AppName:  appName,
		Session:  session,
		Usuario:  updated,
		Success:  "usuario_actualizado",
		EditMode: true,
		IsAdmin:  true,
	}
	h.renderer.RenderPartial(w, "form_usuario", data, http.StatusOK)
}

func (h *UsuarioHandler) renderUsuarioFormError(w http.ResponseWriter, session *middleware.SessionData, errKey string, id int64, fallbackErr string) {
	key := errKey
	if key == "" {
		key = fallbackErr
	}
	data := usuarioFormData{
		Title:    "Editar Usuario",
		AppName:  appName,
		Session:  session,
		Usuario:  &identity.Usuario{ID: id},
		Error:    key,
		EditMode: true,
		IsAdmin:  true,
	}
	h.renderer.RenderPartial(w, "form_usuario", data, http.StatusOK)
}
