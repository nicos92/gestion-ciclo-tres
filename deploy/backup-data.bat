@echo off
REM ============================================================
REM  Detiene el servicio gestion-ciclo-tres, hace backup RAR con
REM  WinRAR de C:\gestion-ciclo-tres\data y vuelve a iniciarlo.
REM  Requiere:
REM    - Ejecutarse como Administrador (o tarea con privilegios)
REM    - WinRAR instalado (Rar.exe o WinRAR.exe en PATH o en
REM      "C:\Program Files\WinRAR\")
REM  Disenado para correr por el Programador de tareas de Windows
REM  (sin prompts ni esperas de consola).
REM  Uso:  backup-data.bat
REM ============================================================
setlocal enabledelayedexpansion

set "SVCCNAME=gestion-ciclo-tres"
set "INSTALL_DIR=C:\gestion-ciclo-tres"
set "DATA_DIR=%INSTALL_DIR%\data"
set "LOG_DIR=%INSTALL_DIR%\logs"
set "BACKUP_DIR=%INSTALL_DIR%\backups"
set "LOG_FILE=%LOG_DIR%\backup.log"
set "HEALTH_URL=http://localhost:8088/healthz"

net session >nul 2>nul || (
  echo [ERROR] Debes ejecutar este script como Administrador.>&2
  exit /b 1
)

REM ---- 1) Detectar WinRAR -------------------------------------
set "RAR_EXE="
where Rar.exe >nul 2>nul && set "RAR_EXE=Rar.exe"
if not defined RAR_EXE ( where WinRAR.exe >nul 2>nul && set "RAR_EXE=WinRAR.exe" )
if not defined RAR_EXE ( if exist "%ProgramFiles%\WinRAR\Rar.exe" set "RAR_EXE=%ProgramFiles%\WinRAR\Rar.exe" )
if not defined RAR_EXE ( if exist "%ProgramFiles%\WinRAR\WinRAR.exe" set "RAR_EXE=%ProgramFiles%\WinRAR\WinRAR.exe" )
if not defined RAR_EXE ( if exist "%ProgramFiles(x86)%\WinRAR\Rar.exe" set "RAR_EXE=%ProgramFiles(x86)%\WinRAR\Rar.exe" )
if not defined RAR_EXE ( if exist "%ProgramFiles(x86)%\WinRAR\WinRAR.exe" set "RAR_EXE=%ProgramFiles(x86)%\WinRAR\WinRAR.exe" )
if not defined RAR_EXE (
  echo [ERROR] No se encontro WinRAR ^(Rar.exe o WinRAR.exe^). Instalalo o agrega su carpeta al PATH.>&2
  exit /b 1
)
echo ==^> WinRAR: %RAR_EXE%

REM ---- 2) Validar servicio y carpeta de datos ------------------
sc query "%SVCCNAME%" >nul 2>nul
if errorlevel 1 (
  echo [ERROR] El servicio %SVCCNAME% no existe. Instalalo antes con install-windows.bat.>&2
  exit /b 1
)
if not exist "%DATA_DIR%" (
  echo [ERROR] No existe la carpeta de datos: %DATA_DIR%.>&2
  exit /b 1
)
if not exist "%LOG_DIR%" mkdir "%LOG_DIR%"

echo [%DATE% %TIME%] ===== Backup iniciado ===== >> "%LOG_FILE%"

REM ---- 3) Detener el servicio ---------------------------------
echo ==^> Deteniendo servicio %SVCCNAME%...
sc stop "%SVCCNAME%" >nul 2>nul
echo ==^> Esperando a que se detenga...
powershell -NoProfile -Command "$svc='%SVCCNAME%'; for ($i=0; $i -lt 30; $i++) { if ((Get-Service -Name $svc).Status -eq 'Stopped') { exit 0 }; Start-Sleep -Seconds 1 }; exit 1" >nul 2>nul
if errorlevel 1 (
  echo [ERROR] El servicio no se detuvo en 30 segundos.>&2
  echo [%DATE% %TIME%] [ERROR] El servicio no se detuvo en 30 segundos >> "%LOG_FILE%"
  exit /b 1
)
set "STATE=STOPPED"
echo ==^> Servicio detenido.

REM ---- 4) Crear backup RAR ------------------------------------
if not exist "%BACKUP_DIR%" mkdir "%BACKUP_DIR%"
for /f %%i in ('powershell -NoProfile -Command "Get-Date -Format yyyyMMdd_HHmmss"') do set "TS=%%i"
set "BACKUP_FILE=%BACKUP_DIR%\data_%TS%.rar"

echo ==^> Creando backup: %BACKUP_FILE%
"%RAR_EXE%" a -y -ep1 -r -idq "%BACKUP_FILE%" "%DATA_DIR%"
if errorlevel 2 (
  echo [ERROR] Fallo la creacion del backup RAR.>&2
  echo [%DATE% %TIME%] [ERROR] Fallo la creacion del backup RAR >> "%LOG_FILE%"
  exit /b 1
)
if errorlevel 1 (
  echo [WARN] El backup RAR termino con advertencias.
)
for %%f in ("%BACKUP_FILE%") do set "BK_SIZE=%%~zf"
echo ==^> Backup OK: !BACKUP_FILE! (!BK_SIZE! bytes)

REM ---- 5) Iniciar el servicio ---------------------------------
echo ==^> Arrancando servicio %SVCCNAME%...
sc start "%SVCCNAME%" >nul 2>nul
echo ==^> Esperando a que arranque...
powershell -NoProfile -Command "$svc='%SVCCNAME%'; for ($i=0; $i -lt 30; $i++) { if ((Get-Service -Name $svc).Status -eq 'Running') { exit 0 }; Start-Sleep -Seconds 1 }; exit 1" >nul 2>nul
if errorlevel 1 (
  echo [ERROR] El servicio no arranco en 30 segundos. Revisar logs en %LOG_DIR%.>&2
  echo [%DATE% %TIME%] [ERROR] El servicio no arranco en 30 segundos >> "%LOG_FILE%"
  exit /b 1
)
set "STATE=RUNNING"
echo ==^> Servicio %SVCCNAME% corriendo.

REM ---- 6) Verificar healthcheck -------------------------------
echo ==^> Verificando %HEALTH_URL%...
set "HEALTH_OK="
for /l %%i in (1,1,10) do (
  powershell -NoProfile -Command "try { $r = Invoke-WebRequest -Uri 'http://localhost:8088/healthz' -UseBasicParsing -TimeoutSec 3; if ($r.StatusCode -eq 200) { exit 0 } else { exit 1 } } catch { exit 1 }" >nul 2>nul
  if not errorlevel 1 (
    set "HEALTH_OK=1"
    goto health_ok
  )
  >nul ping 127.0.0.1 -n 2
)
:health_ok

REM ---- 7) Resumen y log ---------------------------------------
if defined HEALTH_OK goto resumen_ok

echo.
echo Backup completado con advertencia:
echo   - Archivo.......... !BACKUP_FILE!
echo   - Tamano........... !BK_SIZE! bytes
echo   - Servicio......... %SVCCNAME% (%STATE%)
echo   - Healthz.......... %HEALTH_URL% -^> SIN RESPUESTA
echo [%DATE% %TIME%] [WARN] Backup OK: !BACKUP_FILE! (!BK_SIZE! bytes), servicio %SVCCNAME% RUNNING, pero /healthz no respondio >> "%LOG_FILE%"
goto fin

:resumen_ok
echo.
echo Backup completado:
echo   - Archivo.......... !BACKUP_FILE!
echo   - Tamano........... !BK_SIZE! bytes
echo   - Servicio......... %SVCCNAME% (%STATE%)
echo   - Healthz.......... %HEALTH_URL% -^> OK
echo [%DATE% %TIME%] Backup OK: !BACKUP_FILE! (!BK_SIZE! bytes), servicio %SVCCNAME% RUNNING, healthz OK >> "%LOG_FILE%"

:fin
exit /b 0