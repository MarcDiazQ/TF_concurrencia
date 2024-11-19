package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"sync"
)

// Estructura para almacenar información de los productos
type Product struct {
	ID       string  `json:"id"`
	Category string  `json:"category"`
	Stars    float64 `json:"stars"`
}

// Almacenamiento temporal de los productos recibidos
var receivedProducts []Product
var mu sync.Mutex // Para manejar concurrencia al acceder a `receivedProducts`

// Función para manejar conexiones TCP
func tcpReceiver(port string) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println("Error al iniciar el servidor TCP:", err)
		return
	}
	defer listener.Close()

	fmt.Println("Servidor TCP escuchando en el puerto", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error al aceptar conexión:", err)
			continue
		}
		go handleTCPConnection(conn)
	}
}

// Manejar cada conexión TCP
func handleTCPConnection(conn net.Conn) {
	defer conn.Close()

	var products []Product
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&products); err != nil {
		fmt.Println("Error al decodificar datos:", err)
		return
	}

	// Agregar los productos recibidos a la lista global
	mu.Lock()
	receivedProducts = append(receivedProducts, products...)
	mu.Unlock()

	fmt.Println("Productos recibidos y almacenados.")
}

// Función para manejar el servidor HTTP
func htmlHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>Productos Recibidos</title>
	</head>
	<body>
		<h1>Productos Recibidos</h1>
		<table border="1">
			<tr>
				<th>ID</th>
				<th>Categoría</th>
				<th>Estrellas</th>
			</tr>
			{{range .}}
			<tr>
				<td>{{.ID}}</td>
				<td>{{.Category}}</td>
				<td>{{.Stars}}</td>
			</tr>
			{{end}}
		</table>
	</body>
	</html>
	`

	mu.Lock()
	defer mu.Unlock()

	// Generar la página HTML
	t := template.Must(template.New("products").Parse(tmpl))
	t.Execute(w, receivedProducts)
}

func main() {
	// Inicia el receptor TCP en el puerto 8080
	go tcpReceiver("8080")

	// Inicia el servidor HTTP en el puerto 8082 para visualizar los datos
	http.HandleFunc("/", htmlHandler)
	fmt.Println("Servidor HTTP escuchando en el puerto 9090")
	if err := http.ListenAndServe(":9090", nil); err != nil {
		fmt.Println("Error al iniciar el servidor HTTP:", err)
	}
}
