package tarima

import (
	"fmt"
	"strconv"
)

type BarcodeData struct {
	NumeroProducto string
	NumeroTarima   string
	NumeroUsuario  string
	Conservacion   string
	CantidadCajas  int
	Peso           float64
}

func ParseBarcode(raw string) (BarcodeData, error) {
	if len(raw) != 30 {
		return BarcodeData{}, ErrBarcodeLongitud
	}
	if raw[0] != '0' {
		return BarcodeData{}, ErrBarcodePrefix
	}
	if raw[13:17] != "9998" {
		return BarcodeData{}, ErrBarcodeMarker
	}

	cajas, err := strconv.Atoi(raw[21:24])
	if err != nil {
		return BarcodeData{}, fmt.Errorf("parsear cantidad de cajas: %w", err)
	}

	pesoCentavos, err := strconv.Atoi(raw[24:30])
	if err != nil {
		return BarcodeData{}, fmt.Errorf("parsear peso: %w", err)
	}

	return BarcodeData{
		NumeroProducto: raw[1:7],
		NumeroTarima:   raw[7:13],
		Conservacion:   string(raw[17]),
		NumeroUsuario:  raw[18:21],
		CantidadCajas:  cajas,
		Peso:           float64(pesoCentavos) / 100.0,
	}, nil
}
