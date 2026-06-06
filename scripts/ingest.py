import kagglehub
import pandas as pd
import glob

print("Descargando dataset desde Kaggle... (esto puede tardar un poquito la primera vez)")
# Descarga la carpeta entera del dataset usando el "slug" de la URL
dataset_path = kagglehub.dataset_download("ishajangir/crime-data")

# Busca automáticamente cualquier archivo .csv en la carpeta descargada
archivos_csv = glob.glob(f"{dataset_path}/*.csv")

if archivos_csv:
    # Agarramos el primer archivo CSV que encuentre
    ruta_csv = archivos_csv[0]
    print(f"¡Dataset descargado y encontrado en: {ruta_csv}!")
    
    # Ingesta de datos con Pandas
    df = pd.read_csv(ruta_csv)
    
    print("\n¡Ingesta completada, mano! Aquí tienes las primeras 5 filas:")
    display(df.head()) # Usa print(df.head()) si estás en un script normal de Python (.py)
    
else:
    print("Mano, hubo un problema: no se encontró ningún archivo CSV en el dataset descargado.")