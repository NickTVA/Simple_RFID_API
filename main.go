package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/joho/godotenv"
	"net/http"
	"os"
	"rfid_server/database"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type Tag struct {
	Username string
	Tag      string
}

func main() {

	err := godotenv.Load() //by default, it is .env so we don't have to write
	if err != nil {
		fmt.Println("Error has occurred  reading .env file")
	}

	NewRelicAgent, agentInitError := newrelic.NewApplication(newrelic.ConfigFromEnvironment())
	if agentInitError != nil {
		panic(agentInitError)
	}

	// gin.SetMode(gin.ReleaseMode) //optional to not get warning
	route := gin.Default()
	route.Use(nrgin.Middleware(NewRelicAgent))
	nrTxn := NewRelicAgent.StartTransaction("ConnectDatabase")
	database.ConnectDatabase(nrTxn)
	nrTxn.End()
	// NR WARN: function literal segments will be named "function literal" by default
	// declare a function instead to improve segment name generation
	route.GET("/ping", func(context *gin.Context) {
		nrTxn := nrgin.Transaction(context)
		defer nrTxn.StartSegment("function literal").End()

		context.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	route.GET("/get", getTag)

	route.POST("/add", addTag)

	err = route.Run(":8080")
	if err != nil {
		panic(err)
	}

	NewRelicAgent.Shutdown(5 * time.Second)
}

func getTag(ctx *gin.Context) {
	nrTxn := nrgin.Transaction(ctx)
	tagId := ctx.Query("tag")

	s := newrelic.DatastoreSegment{
		Product:            newrelic.DatastorePostgres,
		Collection:         "tags",
		Operation:          "SELECT",
		ParameterizedQuery: "select username from tags where tag=$1",
		QueryParameters: map[string]interface{}{
			"tag": tagId,
		},
		Host:         "postgres",
		PortPathOrID: "5432",
		DatabaseName: "rfid",
	}
	s.StartTime = nrTxn.StartSegmentNow()

	rows, err := database.Db.Query("select username from tags where tag=$1", tagId)
	s.End()
	if err != nil {
		nrTxn.NoticeError(err)
		ctx.AbortWithStatusJSON(400, "Tag is not defined")
		return
	}

	defer rows.Close()

	if rows.Next() {
		username := ""
		rows.Scan(&username)
		print(username)
		ctx.Data(http.StatusOK, "text/plain", []byte(username))
		return

	} else {
		ctx.AbortWithStatusJSON(404, "Tag not found")
		return
	}

}

func addTag(ctx *gin.Context) {
	nrTxn := nrgin.Transaction(ctx)
	body := Tag{}

	restAPIKey := os.Getenv("REST_API_KEY")
	apiKey := ctx.GetHeader("API_KEY")

	if restAPIKey != apiKey {

		nrTxn.NoticeError(errors.New("API key does not match"))
		ctx.AbortWithStatusJSON(401, "Incorrect or missing API Key")
		return
	}

	data, err := ctx.GetRawData()
	if err != nil {
		nrTxn.NoticeError(err)
		ctx.AbortWithStatusJSON(400, "Tag is not defined")
		return
	}
	err = json.Unmarshal(data, &body)
	if err != nil {
		nrTxn.NoticeError(err)
		ctx.AbortWithStatusJSON(400, "Bad Input")
		return
	}
	//use Exec whenever we want to insert update or delete
	//Doing Exec(query) will not use a prepared statement, so lesser TCP calls to the SQL server
	_, err = database.Db.Exec("insert into tags(username,tag) values ($1,$2)", body.Username, body.Tag)
	if err != nil {
		nrTxn.NoticeError(err)
		fmt.Println(err)
		ctx.AbortWithStatusJSON(400, "Couldn't create the new tag.")
	} else {
		ctx.JSON(http.StatusOK, "Tag is successfully created.")
	}

}
