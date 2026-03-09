# csv-server

Serveur HTTP minimal en Go pour démarrer rapidement le projet `csv-server`.

## Prérequis

- Go 1.22+

## Lancer

```bash
go run .
```

Par défaut, le serveur écoute sur `:8080`.

## Endpoint

- `GET /health` → `200 OK` avec le corps `ok`
