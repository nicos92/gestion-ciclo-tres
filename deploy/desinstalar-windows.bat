@echo off
REM ============================================================
REM  Desinstala el servicio Windows gestion-ciclo-tres (NSSM)
REM  Uso:  desinstalar-windows.bat
REM ============================================================
setlocal
set "SVCCNAME=gestion-ciclo-tres"
if "%NSSM_EXE%"=="" (set "NSSM_EXE=nssm")

where "%NSSM_EXE%" >nul 2>nul || (
  echo [ERROR] NSSM no encontrado en el PATH. Setea NSSM_EXE.>&2
  exit /b 1
)

"%NSSM_EXE%" stop "%SVCCNAME%" >nul 2>nul
"%NSSM_EXE%" remove "%SVCCNAME%" confirm
if errorlevel 1 (
  echo [ERROR] No se pudo quitar el servicio.>&2
  exit /b 1
)
echo Servicio %SVCCNAME% desinstalado.
exit /b 0