#!/bin/bash

echo "========================================="
echo "Creando el entorno virtual (venv)..."
echo "========================================="
python3 -m venv venv

echo ""
echo "========================================="
echo "Activando entorno..."
echo "========================================="
source venv/bin/activate
python3 -m pip install --upgrade pip

echo ""
echo "========================================="
echo "Jalando librerías del requirements.txt..."
echo "========================================="
if [ -f requirements.txt ]; then
    pip install -r requirements.txt
else
    echo "[ALERTA] No encuentro requirements.txt."
    echo "Créalo, pon tus librerías y ejecútame de nuevo."
    exit 1
fi

echo ""
echo "========================================="
echo "Registrando el venv como kernel para Jupyter..."
echo "========================================="
pip install ipykernel
python3 -m ipykernel install --user --name=venv_datos --display-name="Python (Venv Datos)"

echo ""
echo "========================================="
echo "¡Todo listo, mano!"
echo "Cuando abras Jupyter, selecciona el kernel 'Python (Venv Datos)'."
echo "========================================="