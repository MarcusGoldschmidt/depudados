# Deputados

Baixa propocições de deputados e extrai seus metadados

## Uso

```bash
go run main.go --help
```

### PLP

```bash
# Obtem arquivos e metadados de uma PLP 
go run main.go --plp 2430143 > output.csv
```

### Todos por ano

```bash
# Obtem arquivos e metadados de uma PLP 
go run main.go --allAno ./proposicoes-2024 > output.csv
```

### Proposicoes

```bash
# Carrega deputados
go run main.go --load-deputados
```

```bash
# usa os deputados ja baixados anteriormente
go run main.go
```

```bash
# Cria arquivo csv com os dados obtidos
go run main.go --generate-csv data.csv
```
