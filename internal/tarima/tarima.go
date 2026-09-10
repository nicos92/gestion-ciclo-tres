package tarima

import "time"

type Tarima struct {
	ID             int64
	CodigoBarras   string
	NumeroProducto string
	NumeroTarima   string
	NumeroUsuario  string
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
