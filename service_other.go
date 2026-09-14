//go:build !windows

package main

// useWindowsService siempre es false fuera de Windows (no hay servicio nativo).
func useWindowsService() bool {
	return false
}

// runAsService no se usa fuera de Windows; existe solo para que main() compile.
func runAsService() {}