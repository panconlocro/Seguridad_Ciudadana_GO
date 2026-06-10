# Reporte de Análisis y Limpieza de Datos

## 1. Introducción
El presente documento detalla el proceso de limpieza y preparación aplicado al conjunto de datos original `Crime_Data_from_2020_to_Present.csv`. Al tratarse de un archivo bastante pesado (cerca de 1 millón de registros), decidimos implementar todo el flujo de procesamiento utilizando Go (Golang) mediante una lectura secuencial (streaming). De esta manera, garantizamos que el script sea rápido y no sature la memoria del sistema. A continuación, se explica el razonamiento detrás de cada transformación realizada en los datos.

## 2. Selección y Reducción de Variables
Al revisar la estructura del dataset original, nos encontramos con varias columnas que, si bien son útiles para el registro administrativo de la policía, introducían mucho ruido para propósitos analíticos. Por ejemplo, la variable `DR_NO` es solo un número de reporte único sin valor predictivo. Otras columnas, como `Mocodes` o los códigos secundarios (`Crm Cd 2`, `Crm Cd 3`, etc.), presentaban un nivel de detalle tan granular y tantos valores vacíos que su uso fragmentaría demasiado cualquier modelo. Asimismo, `Cross Street` resultó ser redundante porque la ubicación del incidente ya estaba capturada en `LOCATION`. Por lo tanto, decidimos eliminar todas estas variables para trabajar con un conjunto de datos mucho más limpio y enfocado.

## 3. Manejo de Fechas y Feature Engineering Temporal
Al observar las fechas en las que ocurrieron los delitos (`DATE OCC`) y las fechas en las que se reportaron (`Date Rptd`), notamos que el sistema las entregaba como simples cadenas de texto. Para que nuestros futuros modelos y visualizaciones puedan aprovechar esta información, decidimos parsear estos textos y extraer métricas clave: creamos las columnas `year`, `month` y `day_of_week`. De este modo, podremos detectar fácilmente estacionalidades (por ejemplo, si hay más robos en verano o los fines de semana).

Además, identificamos que el tiempo que tarda una persona en denunciar un delito es un factor criminológico muy importante. Para capturar esto, calculamos una nueva variable llamada `days_to_report`, que mide la diferencia en días entre la ocurrencia del delito y su reporte oficial. Finalmente, extrajimos la hora exacta (`hour`) a partir del formato militar de `TIME OCC` para facilitar la creación de mapas de calor por franjas horarias.

## 4. Tratamiento de Inconsistencias y Valores Nulos
Durante la exploración de los datos demográficos de las víctimas, encontramos varias inconsistencias que requerían atención:
* **Edades imposibles:** Al revisar `Vict Age`, encontramos víctimas con edades iguales o menores a 0, y algunas superiores a los 99 años. Para manejar este problema y evitar que estos errores de digitación sesguen drásticamente el promedio de edad en nuestros análisis, se decidió convertir todos estos valores atípicos directamente a nulos (NaN).
* **Sexos no estandarizados:** En la columna `Vict Sex`, notamos la presencia de letras sin sentido como `H`, guiones (`-`) o celdas vacías. Para estabilizar la distribución y no tener múltiples categorías de "desconocidos", agrupamos todos estos casos bajo una única etiqueta `X`. Solo se respetaron las categorías claras: `M` (masculino) y `F` (femenino).
* **Armas no reportadas:** Al revisar `Weapon Desc`, vimos muchas celdas en blanco. Asumimos que, en la mayoría de estos casos, la falta de registro indica que no hubo un arma involucrada. Por ello, rellenamos estos espacios con el texto `"NO WEAPON"`, facilitando así la segmentación entre delitos armados y no armados.

## 5. Estandarización de Cadenas de Texto
En variables de ubicación como `LOCATION`, detectamos que algunos registros tenían espacios en blanco adicionales al principio, al final, o dobles espacios entre palabras (ej. `"MAIN  ST"` vs `"MAIN ST"`). De no corregirse, el sistema interpretaría un mismo lugar como dos ubicaciones distintas, inflando artificialmente la cantidad de calles únicas. Para evitar esto, aplicamos una limpieza que eliminó espacios redundantes, dejando el texto totalmente estandarizado.

## 6. Corrección y Análisis Geoespacial
Un problema crítico que encontramos al analizar las coordenadas geográficas fue la presencia de múltiples crímenes registrados exactamente con `LAT = 0` y `LON = 0`. Dejar estos valores intactos situaría equivocadamente estos incidentes en el medio del océano frente a África, arruinando por completo cualquier mapa o algoritmo espacial. 

Para manejar este problema de manera inteligente, diseñamos el script para que agrupe los datos por su respectiva zona policial (`AREA NAME`). Luego, calculamos la **mediana** de las coordenadas válidas de cada zona y reemplazamos los ceros con este valor. Decidimos usar la mediana en lugar del promedio simple porque es una medida mucho más robusta frente a otros posibles errores geográficos en la misma área, asegurando que el crimen se ubique en una posición representativa de su distrito.

## 7. Creación del Perfil de Víctima Identificada
Finalmente, al cruzar la información demográfica, notamos que había un gran bloque de delitos donde la edad, el sexo y la ascendencia de la víctima estaban simultáneamente ausentes o marcados como desconocidos. Estos casos suelen coincidir con delitos contra la propiedad, negocios, o crímenes descubiertos de manera tardía, donde no hay una víctima presencial. 

Para capitalizar este hallazgo, creamos una nueva variable booleana llamada `victim_identified`. Si el registro carece de todos los datos demográficos vitales, se marca como falsa (`false`). Esto nos dará una ventaja enorme más adelante, ya que nos permitirá filtrar rápidamente entre delitos presenciales y no presenciales sin tener que escribir reglas condicionales complejas una y otra vez.

## 8. Formato Final
Como último paso de limpieza, todos los nombres de las columnas resultantes fueron transformados a formato `snake_case` (por ejemplo, de `AREA NAME` a `area_name`). Esto garantiza que el archivo final no genere conflictos de sintaxis al ser cargado en entornos de Python, SQL o herramientas de Business Intelligence.
