# Deputados

Baixa propocições de deputados e extrai seus metadados

## Uso

```bash
go run main.go --help
# Flags
--generate-csv string
criar arquivo csv
--load-deputados
deve carregar todos os deputados
```

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

