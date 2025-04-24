# Simple_RFID_API

PreReqs:
* 
Installation:
* Setup postgres and run createdb in rfid database
* Insert db connection params to .env file
* modify API key if needed
* Set environment variables for New Relic
  * NEW_RELIC_APP_NAME to your app name
  * NEW_RELIC_LICENSE_KEY to your ingest key
* `go run main.go`

Container Install:
* Setup Postgres container
  * Create network to allow cross-container communication
  Example:
  `docker network create go-rfid-network`

  * Set password and network, then Create posrgres db container: 
  ``` 
   docker run -d \
  --rm \
  --name **go-rfid-db** \
  -e POSTGRES_PASSWORD=**Your_Password** \
  -v $(pwd)/postgres-init:/docker-entrypoint-initdb.d \
  -p 5432:5432 \
  --network go-rfid-network \
  postgres:latest 
  ```
  * Note - ensure that you run the above command from the root `Simple_RFID_API` directory
  * Update .env with db host name and password
  ```
  HOST=**go-rfid-db**
  PORT=5432
  DBUSER=postgres
  DB_NAME=rfid
  PASSWORD=**Your_Password**
  REST_API_KEY=A1B2C3D4
  ```
* Build & Run Docker rfid app
  * Execute the following command from the root `Simple_RFID_API` directory
    `docker build -t go-rfid:latest . `      
  * Example Output: 
  ```
  [+] Building 10.0s (12/12) FINISHED                                     docker:desktop-linux
  => [internal] load build definition from Dockerfile                                    0.0s
  => => transferring dockerfile: 370B                                                    0.0s
  => [internal] load metadata for docker.io/library/golang:1.24                          0.6s
  => [auth] library/golang:pull token for registry-1.docker.io                           0.0s
  => [internal] load .dockerignore                                                       0.0s
  => => transferring context: 2B                                                         0.0s
  => [1/6] FROM docker.io/library/golang:1.24@sha256:d9db32125db0c3a680cfb7a1afcaefb89c  0.0s
  => [internal] load build context                                                       0.0s
  => => transferring context: 12.46kB                                                    0.0s
  => CACHED [2/6] WORKDIR /build                                                         0.0s
  => CACHED [3/6] COPY go.mod go.sum ./                                                  0.0s
  => CACHED [4/6] RUN go mod download                                                    0.0s
  => [5/6] COPY . .                                                                      0.1s
  => [6/6] RUN go build -v -o go-rfid-app                                                8.8s
  => exporting to image                                                                  0.5s 
  => => exporting layers                                                                 0.5s 
  => => writing image sha256:371c315982d10c6caf889a86d22a52060fb94ead83ab9be67adefcd88b  0.0s 
  => => naming to docker.io/library/go-rfid:latest                                       0.0s 
                                                                                              
  View build details: docker-desktop://dashboard/build/desktop-linux/desktop-linux/jpag75pf9ls391dzlcuxc0l86
  ```
  * Set the `NEW_RELIC_LICENSE_KEY` and `NEW_RELIC_APP_NAME` variables and start the container with the following command:
  ```
  docker run --rm --name go-rfid-backend \
  --publish 8080:8080 \
  --env-file ./.env \
  -e NEW_RELIC_LICENSE_KEY=**YOUR_NR_LICENSE_KEY** \
  -e NEW_RELIC_APP_NAME=**YOUR_APP_NAME** \
  --network go-rfid-network \
  go-rfid
  ```

  