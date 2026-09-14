@echo off
REM ============================================================
REM  Instala gestion-ciclo-tres como servicio nativo de Windows
REM  (SIN NSSM: se usa el SCM via sc.exe)
REM  Requiere:
REM    - Ejecutarse como Administrador
REM    - Go instalado (para compilar) o un binario ya compilado
REM  Uso:  install-windows.bat [ruta_al_exe]
REM ============================================================
setlocal enabledelayedexpansion

set "SVCCNAME=gestion-ciclo-tres"
set "DISPLAYNAME=Gestion Ciclo Tres"
set "SCRIPT_DIR=%~dp0"
set "REPO_ROOT=%SCRIPT_DIR%.."
set "INSTALL_DIR=C:\gestion-ciclo-tres"
set "DATA_DIR=%INSTALL_DIR%\data"
set "LOG_DIR=%INSTALL_DIR%\logs"
set "EXE_PATH=%~1"

net session >nul 2>nul || (
  echo [ERROR] Debes ejecutar este script como Administrador.>&2
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

REM ---- 2) Quitar el servicio previo si existe ------------------
sc query "%SVCCNAME%" >nul 2>nul
if not errorlevel 1 (
  echo ==^> Removiendo servicio existente...
  sc stop "%SVCCNAME%" >nul 2>nul
  sc delete "%SVCCNAME%" >nul 2>nul
  timeout /t 2 /nobreak >nul
)

REM ---- 3) Crear el servicio ------------------------------------
echo ==^> Creando servicio "%SVCCNAME%"...
sc create "%SVCCNAME%" binPath= "\"%EXE_PATH%\"" start= auto DisplayName= "%DISPLAYNAME%"
if errorlevel 1 (
  echo [ERROR] Fallo al crear el servicio. Revisa la ruta del exe y los permisos.>&2
  exit /b 1
)

sc description "%SVCCNAME%" "Gestion Ciclo Tres: gestion de tarimas"
REM Reinicio automático ante fallas (reset=1dia, 3 reintentos progresivos)
sc failure "%SVCCNAME%" reset= 86400 actions= restart/5000/restart/10000/restart/30000

REM ---- 4) Variables de entorno del servicio -------------------
REM Se escriben en el registro (REG_MULTI_SZ). Son las mismas de internal/config.
echo ==^> Seteando variables de entorno del servicio...
powershell -NoProfile -Command "Set-ItemProperty -Path 'HKLM:\SYSTEM\CurrentControlSet\Services\gestion-ciclo-tres' -Name Environment -Type MultiString -Value @('PORT=:8088'; 'DB_PATH=C:\gestion-ciclo-tres\data\gestion-ciclo-tres.db'; 'TZ=America/Argentina/Buenos_Aires'; 'APP_NAME=Gestion Ciclo Tres'; 'LOG_FILE=C:\gestion-ciclo-tres\logs\gestion-ciclo-tres.log')"
if errorlevel 1 (
  echo [ERROR] Fallo al configurar las variables de entorno del servicio.>&2
  exit /b 1
)

echo ==^> Arrancando el servicio...
sc start "%SVCCNAME%" >nul 2>nul
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
echo Ver estado:    sc query %SVCCNAME%
echo Ver logs:      C:\gestion-ciclo-tres\logs\gestion-ciclo-tres.log  (o Visor de eventos)
echo Parar:         sc stop %SVCCNAME%
echo Arrancar:      sc start %SVCCNAME%
echo Reiniciar:     sc stop %SVCCNAME% ^&^& sc start %SVCCNAME%
echo Desinstalar:   desinstalar-windows.bat
exit /b 0