@echo off
echo =========================================
echo Creando el entorno virtual (venv)...
echo =========================================
python -m venv venv

echo.
echo =========================================
echo Activando entorno...
echo =========================================
call venv\Scripts\activate.bat
python -m pip install --upgrade pip

echo.
echo =========================================
echo Jalando librerias del requirements.txt...
echo =========================================
if exist requirements.txt (
    pip install -r requirements.txt
) else (
    echo [ALERTA] No encuentro requirements.txt.
    echo Crealo, pon tus librerias y ejecutame de nuevo.
    pause
    exit /b
)

echo.
echo =========================================
echo Registrando el venv como kernel para Jupyter...
echo =========================================
pip install ipykernel
python -m ipykernel install --user --name=venv_datos --display-name="Python (Venv Datos)"

echo.
echo =========================================
echo ¡Todo listo, mano!
echo Cuando abras Jupyter, selecciona el kernel "Python (Venv Datos)".
echo =========================================
pause