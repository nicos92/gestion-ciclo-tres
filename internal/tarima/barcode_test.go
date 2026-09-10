package tarima

import (
	"testing"
)

func TestParseBarcodeValido(t *testing.T) {
	bd, err := ParseBarcode("088019700099999981010045045000")
	if err != nil {
		t.Fatalf("ParseBarcode: %v", err)
	}
	if bd.NumeroProducto != "880197" {
		t.Errorf("Producto = %q, want 880197", bd.NumeroProducto)
	}
	if bd.NumeroTarima != "000999" {
		t.Errorf("Tarima = %q, want 000999", bd.NumeroTarima)
	}
	if bd.NumeroUsuario != "010" {
		t.Errorf("Usuario = %q, want 010", bd.NumeroUsuario)
	}
	if bd.Conservacion != "1" {
		t.Errorf("Conservacion = %q, want 1", bd.Conservacion)
	}
	if bd.CantidadCajas != 45 {
		t.Errorf("Cajas = %d, want 45", bd.CantidadCajas)
	}
	if bd.Peso != 450.0 {
		t.Errorf("Peso = %f, want 450.00", bd.Peso)
	}
}

func TestParseBarcodePesoConDecimales(t *testing.T) {
	bd, err := ParseBarcode("088019700099999981010045450055")
	if err != nil {
		t.Fatalf("ParseBarcode: %v", err)
	}
	if bd.Peso != 4500.55 {
		t.Errorf("Peso = %f, want 4500.55", bd.Peso)
	}
}

func TestParseBarcodeLongitudIncorrecta(t *testing.T) {
	_, err := ParseBarcode("08801970009999998101004504500")
	if err != ErrBarcodeLongitud {
		t.Errorf("got %v, want ErrBarcodeLongitud", err)
	}
	_, err = ParseBarcode("0880197000999999810100450450000")
	if err != ErrBarcodeLongitud {
		t.Errorf("got %v, want ErrBarcodeLongitud", err)
	}
}

func TestParseBarcodePrefixIncorrecto(t *testing.T) {
	_, err := ParseBarcode("188019700099999981010045045000")
	if err != ErrBarcodePrefix {
		t.Errorf("got %v, want ErrBarcodePrefix", err)
	}
}

func TestParseBarcodeMarkerIncorrecto(t *testing.T) {
	_, err := ParseBarcode("088019700099999991010045045000")
	if err != ErrBarcodeMarker {
		t.Errorf("got %v, want ErrBarcodeMarker", err)
	}
}

func TestParseBarcodeCeros(t *testing.T) {
	bd, err := ParseBarcode("088019700099999981010000000000")
	if err != nil {
		t.Fatalf("ParseBarcode: %v", err)
	}
	if bd.CantidadCajas != 0 {
		t.Errorf("Cajas = %d, want 0", bd.CantidadCajas)
	}
	if bd.Peso != 0.0 {
		t.Errorf("Peso = %f, want 0.0", bd.Peso)
	}
}

func TestParseBarcodePesosGrandes(t *testing.T) {
	bd, err := ParseBarcode("088019700099999981010045999999")
	if err != nil {
		t.Fatalf("ParseBarcode: %v", err)
	}
	if bd.Peso != 9999.99 {
		t.Errorf("Peso = %f, want 9999.99", bd.Peso)
	}
}

func TestValidateTarimaCompleta(t *testing.T) {
	tarima := &Tarima{
		CodigoBarras:   "088019700099999981010045045000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "010",
		Conservacion:   "1",
		CantidadCajas:  45,
		Peso:           450.00,
		NumeroVenta:    "25-123456",
	}
	if err := ValidateTarima(tarima); err != nil {
		t.Errorf("ValidateTarima: %v", err)
	}
}

func TestValidateTarimaCodigoBarrasRequerido(t *testing.T) {
	tarima := &Tarima{
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroVenta:    "25-123456",
		CantidadCajas:  45,
	}
	if err := ValidateTarima(tarima); err != ErrCodigoBarrasRequerido {
		t.Errorf("got %v, want ErrCodigoBarrasRequerido", err)
	}
}

func TestValidateTarimaNumeroTarimaRequerido(t *testing.T) {
	tarima := &Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroVenta:    "25-123456",
		CantidadCajas:  45,
	}
	if err := ValidateTarima(tarima); err != ErrNumeroTarimaRequerido {
		t.Errorf("got %v, want ErrNumeroTarimaRequerido", err)
	}
}

func TestValidateTarimaVentaFormato(t *testing.T) {
	tarima := &Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroVenta:    "123",
		CantidadCajas:  45,
	}
	if err := ValidateTarima(tarima); err != ErrNumeroVentaFormato {
		t.Errorf("got %v, want ErrNumeroVentaFormato", err)
	}
}

func TestValidateTarimaRangos(t *testing.T) {
	tarima := &Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroVenta:    "25-123456",
		CantidadCajas:  1000,
	}
	if err := ValidateTarima(tarima); err != ErrCantidadCajasRango {
		t.Errorf("got %v, want ErrCantidadCajasRango", err)
	}

	tarima.CantidadCajas = 45
	tarima.Peso = 10000.00
	if err := ValidateTarima(tarima); err != ErrPesoRango {
		t.Errorf("got %v, want ErrPesoRango", err)
	}
}

func TestValidateTarimaProductoMax(t *testing.T) {
	tarima := &Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "8801970",
		NumeroTarima:   "000999",
		NumeroVenta:    "25-123456",
		CantidadCajas:  45,
	}
	if err := ValidateTarima(tarima); err != ErrNumeroProductoMax {
		t.Errorf("got %v, want ErrNumeroProductoMax", err)
	}
}

func TestValidateTarimaUsuarioMax(t *testing.T) {
	tarima := &Tarima{
		CodigoBarras:   "088019700099981010004545000000",
		NumeroProducto: "880197",
		NumeroTarima:   "000999",
		NumeroUsuario:  "0100",
		NumeroVenta:    "25-123456",
		CantidadCajas:  45,
	}
	if err := ValidateTarima(tarima); err != ErrNumeroUsuarioMax {
		t.Errorf("got %v, want ErrNumeroUsuarioMax", err)
	}
}
