import os
import shutil
import glob
import kagglehub

carpeta_destino = "data/raw"
os.makedirs(carpeta_destino, exist_ok=True) 

dataset_path = kagglehub.dataset_download("ishajangir/crime-data")
archivos_csv = glob.glob(f"{dataset_path}/*.csv")

if archivos_csv:
    ruta_origen = archivos_csv[0]
    ruta_final = os.path.join(carpeta_destino, os.path.basename(ruta_origen))
    shutil.copy(ruta_origen, ruta_final)