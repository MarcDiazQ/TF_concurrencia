package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Estructura para representar un producto
type Product struct {
	ID       string  `json:"id"`
	Category string  `json:"category"`
	Stars    float64 `json:"stars"`
}

// Dataset cargado en memoria
var productDataset map[string]Product

// Cargar el dataset desde un archivo CSV
func loadDataset(filePath string) (map[string]Product, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Read() // Omitir encabezado

	products := make(map[string]Product)
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}

		stars, _ := strconv.ParseFloat(record[2], 64)
		products[record[0]] = Product{
			ID:       record[0],
			Category: record[1],
			Stars:    stars,
		}
	}
	return products, nil
}

// Buscar productos de la misma categoría con el mejor rating (excluyendo los originales)
func findBestRecommendations(productIDs []string) []Product {
	categorySet := make(map[string]bool)
	originalProducts := make(map[string]bool)

	// Identificar categorías y marcar los productos originales
	for _, id := range productIDs {
		id = strings.TrimSpace(id)
		product, exists := productDataset[id]
		if exists {
			originalProducts[id] = true
			categorySet[product.Category] = true
		} else {
			fmt.Printf("Product ID not found in dataset: %s\n", id)
		}
	}

	bestProducts := []Product{}
	// Buscar los mejores productos por categoría, excluyendo los productos originales
	for category := range categorySet {
		var bestProduct *Product
		for _, product := range productDataset {
			// Si el producto es de la misma categoría y no es uno de los originales
			if product.Category == category && !originalProducts[product.ID] {
				if bestProduct == nil || product.Stars > bestProduct.Stars {
					bestProduct = &product
				}
			}
		}
		if bestProduct != nil {
			bestProducts = append(bestProducts, *bestProduct)
		}
	}

	return bestProducts
}

// Endpoint para manejar recomendaciones de productos
func productRecommendationHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parsear el cuerpo de la solicitud
	var requestBody struct {
		ProductIDs string `json:"product_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Convertir la lista de IDs en un slice
	productIDs := strings.Split(requestBody.ProductIDs, ",")

	// Buscar los mejores productos por categoría, excluyendo los originales
	bestProducts := findBestRecommendations(productIDs)

	// Imprimir resultados antes de enviarlos
	fmt.Println("Recommended products:", bestProducts)

	// Enviar resultados al puerto 8080
	response, err := sendToServer("localhost:8080", bestProducts)
	if err != nil {
		http.Error(w, "Error connecting to server: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Responder con los datos recibidos
	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

// Enviar resultados al servidor de procesamiento
func sendToServer(address string, bestProducts []Product) ([]byte, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Convertir los mejores productos a JSON
	data, err := json.Marshal(bestProducts)
	if err != nil {
		return nil, err
	}

	// Enviar los datos
	_, err = conn.Write(data)
	if err != nil {
		return nil, err
	}

	// Leer la respuesta del servidor
	var response bytes.Buffer
	_, err = io.Copy(&response, conn)
	if err != nil {
		return nil, err
	}

	return response.Bytes(), nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Configurar los encabezados CORS
		w.Header().Set("Access-Control-Allow-Origin", "*") // Permitir desde cualquier origen
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Manejar preflight requests (OPTIONS)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	// Cargar el dataset al iniciar
	var err error
	productDataset, err = loadDataset("dataset2.csv")
	if err != nil {
		fmt.Println("Error loading dataset:", err)
		return
	}

	// Iniciar servidor HTTP
	http.Handle("/api/recommendations", corsMiddleware(http.HandlerFunc(productRecommendationHandler)))

	fmt.Println("API REST corriendo en :8082")
	if err := http.ListenAndServe(":8082", nil); err != nil {
		fmt.Println("Error al iniciar el servidor:", err)
	}
}