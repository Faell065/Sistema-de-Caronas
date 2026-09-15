package main

import (
	"fmt"
	"projeto_redes/servidor"
)

func main() {
	fmt.Println("   INICIALIZANDO O SERVIDOR VAIJUNTO    ")
	fmt.Println("")

	// Instancia o servidor usando a lógica criada no pacote servidor
	srv := servidor.NovoServidor()
	srv.Iniciar("8080")
}