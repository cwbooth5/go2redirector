@echo off

:: This is here for two purposes.
:: 1. It allows us to define our initial config JSON without tracking the config file itself.
:: 2. It allows comments so users know what fields mean.

set GOCONFIG=go2config.json
set GODB=godb.json

:: if the file doesn't already exist, create a new one with the above contents.
if not exist %GOCONFIG%% (
    echo Writing out new %GOCONFIG%...
    xcopy %GOCONFIG%.init %GOCONFIG%*
) else (
    echo %GOCONFIG% already exists, not overwriting...
)

:: write out a godb.json file with nothing inside
if not exist %GODB%% (
    echo Writing out new initial %GODB%...
    xcopy %GODB%.init %GODB%*
) else (
    echo %GODB% already exists, not overwriting...
)

echo Setup complete.