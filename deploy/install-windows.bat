@echo off
REM ============================================================
REM  Instala gestion-ciclo-tres como servicio Windows via NSSM
REM  Requiere:
REM    - Go instalado (para compilar) o un binario ya compilado
REM    - NSSM (https://nssm.cc) disponible en el PATH o en NSSM_EXE
REM  Uso:  install-windows.bat [ruta_al_exe]
REM ============================================================
setlocal enabledelayedexpansion

set "SVCCNAME=gestion-ciclo-tres"
set "DISPLAYNAME=Gestión Ciclo Tres"
set "SCRIPT_DIR=%~dp0"
set "REPO_ROOT=%SCRIPT_DIR%.."
set "INSTALL_DIR=C:\gestion-ciclo-tres"
set "DATA_DIR=%INSTALL_DIR%\data"
set "LOG_DIR=%INSTALL_DIR%\logs"
set "EXE_PATH=%~1"

if "%NSSM_EXE%"=="" (set "NSSM_EXE=nssm")
where "%NSSM_EXE%" >nul 2>nul || (
  echo [ERROR] NSSM no encontrado en el PATH. Descargalo de https://nssm.cc o setea NSSM_EXE.>&2
  exit /b 1
)

echo ==^> Creando carpetas...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%DATA_DIR%"   mkdir "%DATA_DIR%"
if not exist "%LOG_DIR%"    mkdir "%LOG_DIR%"

REM ---- 1) Compilar o validar el binario -----------------------
if "%EXE_PATH%"=="" (
  echo ==^> Compilando binario para Windows...
  pushd "%REPO_ROOT%"
  set "CGO_ENABLED=0"
  set "GOOS=windows"
  set "GOARCH=amd64"
  go build -trimpath -ldflags="-s -w" -o "%INSTALL_DIR%\gestion-ciclo-tres.exe" .
  if errorlevel 1 (
    echo [ERROR] Fallo la compilacion. Revisa que exista go.mod en %REPO_ROOT%.>&2
    popd
    exit /b 1
  )
  popd
  set "EXE_PATH=%INSTALL_DIR%\gestion-ciclo-tres.exe"
) else (
  if not exist "%EXE_PATH%" (
    echo [ERROR] El ejecutable no existe: %EXE_PATH%>&2
    exit /b 1
  )
)

echo ==^> Instalando servicio "%SVCCNAME%"...
"%NSSM_EXE%" install "%SVCCNAME%" "%EXE_PATH%"
if errorlevel 1 exit /b 1

"%NSSM_EXE%" set "%SVCCNAME%" DisplayName "%DISPLAYNAME%"
"%NSSM_EXE%" set "%SVCCNAME%" Description "Gestión Ciclo Tres"
"%NSSM_EXE%" set "%SVCCNAME%" AppDirectory "%INSTALL_DIR%"
"%NSSM_EXE%" set "%SVCCNAME%" Start SERVICE_AUTO_START
"%NSSM_EXE%" set "%SVCCNAME%" AppStdout "%LOG_DIR%\stdout.log"
"%NSSM_EXE%" set "%SVCCNAME%" AppStderr "%LOG_DIR%\stderr.log"
"%NSSM_EXE%" set "%SVCCNAME%" AppRotateFiles 1
"%NSSM_EXE%" set "%SVCCNAME%" AppRotateOnline 1

echo ==^> Seteando variables de entorno del servicio...
REM Cada KEY=VALUE va como argumento separado; los valores con espacios van entre comillas.
"%NSSM_EXE%" set "%SVCCNAME%" AppEnvironmentExtra ^
  "PORT=:8088" ^
  "DB_PATH=%DATA_DIR%\gestion-ciclo-tres.db" ^
  "TZ=America/Argentina/Buenos_Aires" ^
  "APP_NAME=Gestión Ciclo Tres"

echo ==^> Arrancando el servicio...
"%NSSM_EXE%" start "%SVCCNAME%"
if errorlevel 1 (
  echo [ERROR] No se pudo iniciar el servicio. Revisar logs en %LOG_DIR%.>&2
  exit /b 1
)

echo.
echo Servicio instalado y corriendo:
echo   - Servicio................. %SVCCNAME%
echo   - Ejecutable............... %EXE_PATH%
echo   - Base de datos............ %DATA_DIR%\gestion-ciclo-tres.db
echo   - Logs..................... %LOG_DIR%
echo   - URL...................... http://localhost:8088
echo Ver estado:    nssm status %SVCCNAME%
echo Ver logs:      nssm log %SVCCNAME%   (o revisar %LOG_DIR%)
echo Parar:         nssm stop %SVCCNAME%
echo Reiniciar:     nssm restart %SVCCNAME%
echo Desinstalar:   desinstalar-windows.bat   (o nssm remove %SVCCNAME% confirm)
exit /b 0
