# Beispiel-REST-API in Go

Diese beispielhafte REST-API in Go bietet einen einzigen, resourcenbasierten und offenen Endpunkt (ohne Authentifizierung), der eine Resource mit Namen und Nachricht verwaltet. die API erfüllt keinen praktischen Nutzen, sie dient lediglich einer Demonstration meiner Fähigkeiten.

# Disclaimer

Die API, der Endpunkt und jede abgelegte Resource ist öffentlich zugänglich und kann von jedem Benutzer gelesen, verändert, oder gelöscht werden. Ich kann nicht für die Inhalte haften. Sollte ich einen Missbrauch feststellen, dann wird die API entsprechend eingeschränkt oder abgeschaltet.

# Nutzung

## Endpunkte

### /objects(/:id)
`https://go.point-pioneers.net/objects`

Methoden:
- `GET` - Listet alle Objekte auf oder ein einzelnes Objekt, wenn eine ID angegeben ist.
- `POST` - Erstellt ein neues Objekt.
- `PUT` - Aktualisiert ein bestehendes Objekt.
- `DELETE` - Löscht ein bestehendes Objekt.

Beispielobjekt:
```json
{
  "name": "Beispielname",
  "message": "Beispielnachricht"
}
```

### /healthz
`https://go.point-pioneers.net/healthz`

Methoden:
- `GET` - Prüft den Gesundheitszustand der API.

## Entwicklung

Entwickler sind eingeladen, den Quellcode zu verändern und Pull Requests einzureichen. Sobald ich den Pull Request annehme, übernimmt das automatische Deployment mit Docker die Aktualisierung der API.

## Beispiele (curl)

Objekt erstellen:

```bash
curl -X POST https://go.point-pioneers.net/objects \
  -H "Content-Type: application/json" \
  -d '{"name": "Hans", "message": "Hello world"}'
```

Alle Objekte auflisten:

```bash
curl https://go.point-pioneers.net/objects
```

Ein Objekt abrufen:

```bash
curl https://go.point-pioneers.net/objects/1
```

Ein Objekt aktualisieren:

```bash
curl -X PUT https://go.point-pioneers.net/objects/1 \
  -H "Content-Type: application/json" \
  -d '{"name": "Hans", "message": "Hallo Welt"}'
```

Ein Objekt löschen:

```bash
curl -X DELETE https://go.point-pioneers.net/objects/1
```

Gesundheitszustand prüfen:

```bash
curl https://go.point-pioneers.net/healthz
```
