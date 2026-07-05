package training

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const versionModelo = 1

type ModeloPersistido struct {
	Version       int                 `json:"version"`
	Tipo          string              `json:"tipo"`
	Algoritmo     string              `json:"algoritmo"`
	Features      []string            `json:"features"`
	EntrenadoEn   time.Time           `json:"entrenado_en"`
	Configuracion ConfigEntrenamiento `json:"configuracion"`
	Modelo1       *Modelo1            `json:"modelo1,omitempty"`
	Modelo2       *Modelo2            `json:"modelo2,omitempty"`
	Modelo3       *Modelo3            `json:"modelo3,omitempty"`
}

func GuardarModelo(path string, modelo *ModeloPersistido) error {
	if modelo == nil {
		return fmt.Errorf("no hay modelo para guardar")
	}
	if err := validarModeloPersistido(modelo); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("no se pudo crear el directorio del modelo: %w", err)
	}
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("no se pudo crear %q: %w", path, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(modelo); err != nil {
		return fmt.Errorf("no se pudo serializar el modelo: %w", err)
	}
	fmt.Printf("✔ Modelo guardado en %s\n", path)
	return nil
}

func CargarModelo(path string) (*ModeloPersistido, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir el modelo %q: %w", path, err)
	}
	defer file.Close()

	var modelo ModeloPersistido
	if err := json.NewDecoder(file).Decode(&modelo); err != nil {
		return nil, fmt.Errorf("modelo JSON inválido: %w", err)
	}
	if modelo.Version != versionModelo {
		return nil, fmt.Errorf("versión de modelo no soportada: %d", modelo.Version)
	}
	if err := validarModeloPersistido(&modelo); err != nil {
		return nil, err
	}
	return &modelo, nil
}

func validarModeloPersistido(modelo *ModeloPersistido) error {
	if len(modelo.Features) == 0 {
		return fmt.Errorf("el modelo no declara sus features")
	}
	switch modelo.Tipo {
	case "model1":
		if modelo.Modelo1 == nil || modelo.Modelo1.NumArboles < 1 ||
			len(modelo.Modelo1.Arboles) != modelo.Modelo1.NumArboles {
			return fmt.Errorf("el archivo no contiene un model1 válido")
		}
	case "model2":
		if modelo.Modelo2 == nil || modelo.Modelo2.NumArboles < 1 ||
			len(modelo.Modelo2.ArbolesLat) != modelo.Modelo2.NumArboles ||
			len(modelo.Modelo2.ArbolesLon) != modelo.Modelo2.NumArboles {
			return fmt.Errorf("el archivo no contiene un model2 válido")
		}
	case "model3":
		if modelo.Modelo3 == nil || modelo.Modelo3.NumArboles < 1 ||
			len(modelo.Modelo3.Arboles) != modelo.Modelo3.NumArboles {
			return fmt.Errorf("el archivo no contiene un model3 válido")
		}
	default:
		return fmt.Errorf("tipo de modelo desconocido: %q", modelo.Tipo)
	}
	return nil
}
