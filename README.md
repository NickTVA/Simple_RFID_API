# Simple_RFID_API


## Local Installation:

### Prerequisites:
* Install Golang environment
* Install and setup postgres database
  * run `postgres-init/createdb.sql` in rfid database
### Configure & Run
* Update Environment Variables 
  * Modify database connection parameters in .env file or in Local ENV variables
  * Update API key if necessary
* Set environment variables for New Relic
  * NEW_RELIC_APP_NAME to your app name
  * NEW_RELIC_LICENSE_KEY to your ingest key
* `go run main.go`

## Container Install:

### Prerequisites:
* Install Docker

* Setup Postgres container
  * Create network to allow cross-container communication
  Example:
  `docker network create go-rfid-network`

  * Set password and network, then Create posrgres db container: 
  ``` 
   docker run -d \
  --rm \
  --name go-rfid-db \
  -e POSTGRES_PASSWORD=**Your_Password** \
  -v $(pwd)/postgres-init:/docker-entrypoint-initdb.d \
  -p 5432:5432 \
  --network go-rfid-network \
  postgres:latest 
  ```
  * Note - ensure that you run the above command from the root `Simple_RFID_API` directory
### Build & Deploy
* Update .env with db host name and password
  ```
  HOST=go-rfid-db
  PORT=5432
  DBUSER=postgres
  DB_NAME=rfid
  PASSWORD=Your_Password
  REST_API_KEY=A1B2C3D4
  ```
* Build & Run Docker rfid app
  * Execute the following command from the root `Simple_RFID_API` directory
  ```
    `docker build -t go-rfid:latest . `      
  ```
  * Set the `NEW_RELIC_LICENSE_KEY` and `NEW_RELIC_APP_NAME` variables and start the container with the following command:
  ```
  docker run --rm --name go-rfid-backend \
  --publish 8080:8080 \
  --env-file ./.env \
  -e NEW_RELIC_LICENSE_KEY=**YOUR_NR_LICENSE_KEY** \
  -e NEW_RELIC_APP_NAME=YOUR_APP_NAME \
  --network go-rfid-network \
  go-rfid
  ```


