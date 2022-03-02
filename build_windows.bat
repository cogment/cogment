@echo off

set ROOT_DIR=%~dp0

set BUILD_DIR="%ROOT_DIR%/build/windows_amd64"
md %BUILD_DIR%

set INSTALL_DIR="%ROOT_DIR%/install/windows_amd64"
md %INSTALL_DIR%

cd %BUILD_DIR%
cmake -DCMAKE_BUILD_TYPE="Release" -DCMAKE_INSTALL_PREFIX="%INSTALL_DIR%" -DCOGMENT_EMBEDS_ORCHESTRATOR=ON -DCOGMENT_OS=windows -DCOGMENT_ARCH=amd64 "%ROOT_DIR%"
IF %ERRORLEVEL% NEQ 0 (
  echo "--==-- Error while running cmake --==--"
  exit %ERRORLEVEL%
)
cmake --build . --config release
IF %ERRORLEVEL% NEQ 0 (
  echo "--==-- Error while running the build --==--"
  exit %ERRORLEVEL%
)
cmake --install . --config release
IF %ERRORLEVEL% NEQ 0 (
  echo "--==-- Error while running the installation --==--"
  exit %ERRORLEVEL%
)
