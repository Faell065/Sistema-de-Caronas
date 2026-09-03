package main
import "fmt"
import "bufio"
import "os"

func main() {
	

}

func teste(){
	fmt.Println("Hello, World!")
	nome := print("Seu nome:");
	println("Olá,", nome);
	var num1,num2,num3 int;
	num1 = 3;
	num2 = 9;
	num3 = num1 + num2;
	fmt.Println( num1, "+", num2, "=", num3);

	for i := 0; i < 10; i++ {
		num1 ++;
		num2 -= i;
		num3 = num1 + num2;
		fmt.Println( num1, "+", num2, "=", num3);

	}
}

func print(pergunta string) string {
	fmt.Println(pergunta);
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text();

}

// Classes para o Sistema
type Trecho struct {
	Origem string
	Destino string
	AssentosTotais int
	AssentosOcupados int

}

type Rota struct {
	ID string
	MotoristaID string
	Trechos []Trecho // Lista dinamica (slice)
}

