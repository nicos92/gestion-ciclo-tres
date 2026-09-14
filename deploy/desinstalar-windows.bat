@echo off
REM ============================================================
REM  Desinstala el servicio nativo de Windows gestion-ciclo-tres
REM  Requiere ejecutarse como Administrador.
REM  Uso:  desinstalar-windows.bat
REM ============================================================
setlocal
set "SVCCNAME=gestion-ciclo-tres"

net session >nul 2>nul || (
  echo [ERROR] Debes ejecutar este script como Administrador.>&2
  exit /b 1
)

sc query "%SVCCNAME%" >nul 2>nul
if errorlevel 1 (
  echo El servicio %SVCCNAME% no existe.
  exit /b 0
)

echo ==^> Deteniendo servicio...
sc stop "%SVCCNAME%" >nul 2>nul
timeout /t 2 /nobreak >nul

echo ==^> Eliminando servicio...
sc delete "%SVCCNAME%"
if errorlevel 1 (
  echo [ERROR] No se pudo quitar el servicio.>&2
  exit /b 1
)
echo Servicio %SVCCNAME% desinstalado.
exit /b 0