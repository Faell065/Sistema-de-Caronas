package dominio

import (
	"path/filepath"
	"testing"
)

func TestPersistenciaIsoladaEntreEmpresas(t *testing.T) {
	raiz := t.TempDir()
	a := filepath.Join(raiz, "a")
	b := filepath.Join(raiz, "b")
	ga := NovoGerenciadorComDiretorio(a)
	gb := NovoGerenciadorComDiretorio(b)
	id, err := ga.CadastrarUsuario("Pessoa A", "teste", "passageiro")
	if err != nil {
		t.Fatal(err)
	}
	if len(gb.Usuarios) != 0 {
		t.Fatal("B recebeu os dados de A")
	}
	ga2 := NovoGerenciadorComDiretorio(a)
	gb2 := NovoGerenciadorComDiretorio(b)
	if _, ok := ga2.Usuarios[id]; !ok {
		t.Fatal("A não recuperou seus dados")
	}
	if len(gb2.Usuarios) != 0 {
		t.Fatal("B carregou dados de A")
	}
}
