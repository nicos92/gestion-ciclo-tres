package render

import (
	"io/fs"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testTemplatesFS apunta a la raíz del repositorio, igual que el embed de main.go.
func testTemplatesFS(t *testing.T) fs.FS {
	t.Helper()
	return os.DirFS(filepath.Join("..", "..", ".."))
}

func TestRenderPartial_PartialInvocaOtroPartial(t *testing.T) {
	r, err := New(testTemplatesFS(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	r.RenderPartial(rec, "form_tarimas", map[string]any{
		"CampoProducto": map[string]string{"Numero": "2", "Nombre": "SERVICIO EN SEGURIDAD E HIGIENE"},
	}, 200)

	body := rec.Body.String()
	for _, want := range []string{`id="nombreProducto"`, "SERVICIO EN SEGURIDAD E HIGIENE", "readonly"} {
		if !strings.Contains(body, want) {
			t.Errorf("el fragmento no contiene %q", want)
		}
	}
}

// El nombre del producto es informativo: sin name no viaja en el POST.
func TestRenderPartial_CampoNombreProductoNoSeEnvia(t *testing.T) {
	r, err := New(testTemplatesFS(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	r.RenderPartial(rec, "campo_nombre_producto", map[string]string{"Numero": "111111"}, 200)

	body := rec.Body.String()
	if strings.Contains(body, `name="nombreProducto"`) {
		t.Error("el campo de nombre de producto no debe tener atributo name")
	}
	if !strings.Contains(body, "Sin nombre asignado en el catálogo") {
		t.Error("falta el placeholder para un producto sin nombre en el catálogo")
	}
}

func TestFormatNumber(t *testing.T) {
	cases := []struct {
		input interface{}
		want  string
	}{
		{0, "0"},
		{2, "2"},
		{1000, "1.000"},
		{1234, "1.234"},
		{1234567, "1.234.567"},
		{int64(12345678), "12.345.678"},
		{float64(999.5), "999"},
		{-1234, "-1.234"},
		{"abc", "abc"},
	}
	for _, c := range cases {
		got := formatNumber(c.input)
		if got != c.want {
			t.Errorf("formatNumber(%v) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestFormatFloat(t *testing.T) {
	cases := []struct {
		input float64
		want  string
	}{
		{0, "0.00"},
		{450.00, "450.00"},
		{9999.99, "9999.99"},
		{12.5, "12.50"},
	}
	for _, c := range cases {
		got := formatFloat(c.input)
		if got != c.want {
			t.Errorf("formatFloat(%v) = %q, want %q", c.input, got, c.want)
		}
	}
}
