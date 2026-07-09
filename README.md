# Clima por CEP

API em Go que recebe um CEP, resolve a cidade via ViaCEP e retorna a temperatura atual via WeatherAPI.

## Endpoint

`GET /?cep=01001000`

Resposta:

```json
{
  "temp_C": 28.5,
  "temp_F": 83.3,
  "temp_K": 301.65
}
```

## Erros

- `422 invalid zipcode`
- `404 can not find zipcode`

## Rodando localmente com Docker

```bash
docker build -t clima-cep .
docker run --rm -p 8080:8080 -e WEATHER_API_KEY=seu_token clima-cep
```

O serviço depende da `WEATHER_API_KEY` para consultar a WeatherAPI.

## Testando com curl

Com a aplicação rodando em `http://localhost:8080`:

```bash
curl "http://localhost:8080/?cep=01001000"
```

Resposta de sucesso:

```json
{"temp_C":28.5,"temp_F":83.3,"temp_K":301.65}
```

CEP inválido:

```bash
curl "http://localhost:8080/?cep=123"
```

CEP não encontrado:

```bash
curl "http://localhost:8080/?cep=00000000"
```

## Testes

```bash
go test ./...
```

