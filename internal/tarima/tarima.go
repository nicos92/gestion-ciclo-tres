package tarima

import (
	"errors"
	"regexp"
	"time"
)

var (
	ErrBarcodeLongitud       = errors.New("el código de barras debe tener exactamente 30 dígitos")
	ErrBarcodePrefix         = errors.New("el código de barras debe iniciar con 0")
	ErrBarcodeMarker         = errors.New("el código de barras no pertenece a una tarima (se esperaba 9998)")
	ErrCodigoBarrasRequerido = errors.New("el código de barras es obligatorio")
	ErrNumeroTarimaRequerido = errors.New("el número de tarima es obligatorio")
	ErrNumeroVentaFormato    = errors.New("el número de venta debe seguir el formato XX-XXXXXX")
	ErrNumeroProductoMax     = errors.New("el número de producto no puede tener más de 6 dígitos")
	ErrNumeroTarimaMax       = errors.New("el número de tarima no puede tener más de 6 dígitos")
	ErrNumeroUsuarioMax      = errors.New("el número de usuario no puede tener más de 3 dígitos")
	ErrCantidadCajasRango    = errors.New("la cantidad de cajas debe ser entre 1 y 999")
	ErrPesoRango             = errors.New("el peso debe ser entre 0 y 9999.99")
	ErrNumeroVentaMax        = errors.New("el número de venta no puede tener más de 9 caracteres")
	ErrCodigoBarrasDuplicado = errors.New("ya existe una tarima registrada con este código de barras")
	ErrBarcodeNoCoincide     = errors.New("el código de barras no coincide con los campos del formulario")
	ErrTarimaNoEncontrada    = errors.New("tarima no encontrada")
)

var ventaRegex = regexp.MustCompile(`^\d{2}-\d{6}$`)

type Tarima struct {
	ID             int64
	CodigoBarras   string
	NumeroProducto string
	NumeroTarima   string
	NumeroUsuario  string
	Conservacion   string
	CantidadCajas  int
	Peso           float64
	NumeroVenta    string
	Descripcion    string
	IDUsuario      *int64
	FechaRegistro  time.Time
	Fecha          time.Time
	Legajo         string
	NombreUsuario  string
}

type FiltrosTarima struct {
	NumeroProducto  string
	NumeroTarima    string
	NumeroUsuario   string
	NumeroVenta     string
	FechaRegistro   string
	Legajo          string
	NombreUsuario   string
	CantidadCajasMin *int
	PesoMin          *float64
}

func (f FiltrosTarima) HasFilters() bool {
	if f.NumeroProducto != "" || f.NumeroTarima != "" || f.NumeroUsuario != "" ||
		f.NumeroVenta != "" || f.FechaRegistro != "" || f.Legajo != "" ||
		f.NombreUsuario != "" {
		return true
	}
	if f.CantidadCajasMin != nil || f.PesoMin != nil {
		return true
	}
	return false
}

func ValidateTarima(t *Tarima) error {
	if t.CodigoBarras == "" {
		return ErrCodigoBarrasRequerido
	}
	if len(t.CodigoBarras) != 30 {
		return ErrBarcodeLongitud
	}
	if t.CodigoBarras[0] != '0' {
		return ErrBarcodePrefix
	}
	if t.CodigoBarras[13:17] != "9998" {
		return ErrBarcodeMarker
	}
	if t.NumeroTarima == "" {
		return ErrNumeroTarimaRequerido
	}
	if len(t.NumeroProducto) > 6 {
		return ErrNumeroProductoMax
	}
	if len(t.NumeroTarima) > 6 {
		return ErrNumeroTarimaMax
	}
	if len(t.NumeroUsuario) > 3 {
		return ErrNumeroUsuarioMax
	}
	if t.CantidadCajas < 1 || t.CantidadCajas > 999 {
		return ErrCantidadCajasRango
	}
	if t.Peso < 0 || t.Peso > 9999.99 {
		return ErrPesoRango
	}
	if len(t.NumeroVenta) > 9 {
		return ErrNumeroVentaMax
	}
	if !ventaRegex.MatchString(t.NumeroVenta) {
		return ErrNumeroVentaFormato
	}
	return nil
}
