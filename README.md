# Simple_RFID_API

Installation:
* Setup postgres and run createdb in rfid database
* Insert db connection params to .env file
* modify API key if needed
* Set environment variables for New Relic
  * NEW_RELIC_APP_NAME to your app name
  * NEW_RELIC_LICENSE_KEY to your ingest key
* `go run main.go`