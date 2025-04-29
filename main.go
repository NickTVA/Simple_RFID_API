package main

import (
	"encoding/json"
	"errors"
	"github.com/joho/godotenv"
	"net/http"
	"os"
	"rfid_server/database"
	"time"
	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/integrations/nrgin"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/rs/zerolog"
	"github.com/newrelic/go-agent/v3/integrations/nrzerolog"
	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/zerologWriter"
)	

type Tag struct {
    Username string    `json:"username"`
    Tag      string    `json:"tag"`
    Expire   time.Time `json:"expire"`
}
var logger zerolog.Logger
var writer zerologWriter.ZerologWriter

func main() {

	// Optional:Create a file for logging (Default logs to stdout)
	// file, err := os.OpenFile(
    //     "rfid-backend.log",
    //     os.O_APPEND|os.O_CREATE|os.O_WRONLY,
    //     0664,
    // )
    // if err != nil {
    //     panic(err)
    // }

    // defer file.Close()
    
	//validate time elements
	var myTime = time.Now()
	println("Current time: ", myTime.String())	
	

	err := godotenv.Load() //by default, it is .env so we don't have to write
	if err != nil {
			println("Error loading .env file")
	} 

	NewRelicAgent, agentInitError := newrelic.NewApplication(
		newrelic.ConfigFromEnvironment(),
		newrelic.ConfigDebugLogger(os.Stdout),
		nrzerolog.ConfigLogger(&logger),
		newrelic.ConfigInfoLogger(os.Stdout),
		newrelic.ConfigAppLogDecoratingEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)
	if agentInitError != nil {
		panic(agentInitError)
	}
	NewRelicAgent.WaitForConnection(5 * time.Second)

	writer = zerologWriter.New(os.Stdout,NewRelicAgent)
	logger = zerolog.New(writer)
	logger.Info().Msg("Starting RFID Backend")


	// gin.SetMode(gin.ReleaseMode) //optional to not get warning
	route := gin.Default()
	route.Use(nrgin.Middleware(NewRelicAgent))
	nrTxn := NewRelicAgent.StartTransaction("ConnectDatabase")
	defer nrTxn.End()
	txnLogger := logger.Output(writer.WithTransaction(nrTxn))
	database.ConnectDatabase(nrTxn)
	txnLogger.Info().Msg("Connected to database")

	// NR WARN: function literal segments will be named "function literal" by default
	// declare a function instead to improve segment name generation
	route.GET("/ping", func(context *gin.Context) {
		nrTxn := nrgin.Transaction(context)
		txnLogger := logger.Output(writer.WithTransaction(nrTxn))
		 nrTxn.StartSegment("function literal")

		context.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
		txnLogger.Trace().Msg("Ping endpoint hit")

	})

	route.GET("/get", getTag)

	route.POST("/add", addTag)

	route.GET("/del", deleteTag)

	err = route.Run(":8080")
	if err != nil {
		panic(err)
	}

	NewRelicAgent.Shutdown(5 * time.Second)
}

func getTag(ctx *gin.Context) {
	nrTxn := nrgin.Transaction(ctx)
	tagId := ctx.Query("tag")
	nrTxn.AddAttribute("tagId",tagId)
	txnLogger := logger.Output(writer.WithTransaction(nrTxn))
	txnLogger.Trace().Msg("Get Tag endpoint hit")

	if isUserExpired(tagId) {
		txnLogger.Info().Msg("User Access expired")
		ctx.AbortWithStatusJSON(403, "Access Expired")
		return
	}

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
		txnLogger.Info().Msg("Found User: " + username + " for tag: " + tagId)
		nrTxn.AddAttribute("user",username)
		ctx.Data(http.StatusOK, "text/plain", []byte(username))
		return

	} else {
		ctx.AbortWithStatusJSON(404, "Tag not found")
		return
	}

}

func addTag(ctx *gin.Context) {
    nrTxn := nrgin.Transaction(ctx)
	txnLogger := logger.Output(writer.WithTransaction(nrTxn))
    body := Tag{}
	nrTxn.AddAttribute("tagId",body.Tag)
	nrTxn.AddAttribute("user",body.Username)
	nrTxn.AddAttribute("expireDate",body.Expire)

    restAPIKey := os.Getenv("REST_API_KEY")
    apiKey := ctx.GetHeader("API_KEY")

    if restAPIKey != apiKey {
        nrTxn.NoticeError(errors.New("API key does not match"))
		txnLogger.Error().Err(errors.New("API key does not match")).Msg("Incorrect or missing API Key")
        ctx.AbortWithStatusJSON(401, "Incorrect or missing API Key")
        return
    }

    // Parse the request body
    data, err := ctx.GetRawData()
    if err != nil {
        nrTxn.NoticeError(err)	
		txnLogger.Error().Err(err).Msg("Error getting raw data.  Tag is not defined")
        ctx.AbortWithStatusJSON(400, "Tag is not defined")
        return
    }
    err = json.Unmarshal(data, &body)
    if err != nil {
        nrTxn.NoticeError(err)
		txnLogger.Error().Err(err).Msg("Error unmarshalling JSON")
        ctx.AbortWithStatusJSON(400, "Bad Input")
        return
    }
    // Set default values if fields are not provided
    if body.Username == ""  {
        body.Username = "guest_"+body.Tag // Default username
        txnLogger.Debug().Msg("Username not provided. Setting default username: default_user")
    }
    if body.Expire.IsZero() {
        body.Expire = time.Now().Add(24 * time.Hour) // Default expiration: 24 hours from now
        txnLogger.Debug().Msg("Expire not provided. Setting default expiration to 24 hours from now")
    }

    // Check if the tag already exists
    var existingUsername string
    err = database.Db.QueryRow("select username from tags where tag=$1", body.Tag).Scan(&existingUsername)
    if err == nil {
        // Tag already exists
		txnLogger.Info().Str("tag", body.Tag).Msg("Tag already exists")
        ctx.AbortWithStatusJSON(409, "Tag already exists")
        return
    } else if err != nil && err.Error() != "sql: no rows in result set" {
        // Handle unexpected database errors
        nrTxn.NoticeError(err)
		txnLogger.Error().Err(err).Msg("Error checking if tag exists")
        ctx.AbortWithStatusJSON(500, "Internal Server Error")
        return
    }

    // Insert the new tag
    _, err = database.Db.Exec("insert into tags(username,tag,expire) values ($1,$2,$3)", body.Username, body.Tag, body.Expire)
    if err != nil {
        nrTxn.NoticeError(err)
        txnLogger.Error().Err(err).Msg("Error inserting new tag")
        ctx.AbortWithStatusJSON(400, "Couldn't create the new tag.")
    } else {
		txnLogger.Debug().Str("tag", body.Tag).Msg("Tag is successfully created")
        ctx.JSON(http.StatusOK, "User Added")
    }
}

func deleteTag(ctx *gin.Context) {
    nrTxn := nrgin.Transaction(ctx)
    tagId := ctx.Query("tag")
	nrTxn.AddAttribute("tagId",tagId)
	txnLogger := logger.Output(writer.WithTransaction(nrTxn))
	txnLogger.Trace().Msg("Delete Tag endpoint hit")

    // Define a New Relic DatastoreSegment for monitoring
    s := newrelic.DatastoreSegment{
        Product:            newrelic.DatastorePostgres,
        Collection:         "tags",
        Operation:          "DELETE",
        ParameterizedQuery: "delete from tags where tag=$1",
        QueryParameters: map[string]interface{}{
            "tag": tagId,
        },
        Host:         "postgres",
        PortPathOrID: "5432",
        DatabaseName: "rfid",
    }
    s.StartTime = nrTxn.StartSegmentNow()

    // Execute the DELETE query
    result, err := database.Db.Exec("delete from tags where tag=$1", tagId)
    s.End()
    if err != nil {
        nrTxn.NoticeError(err)
		txnLogger.Error().Err(err).Msg("Error deleting tag")
        ctx.AbortWithStatusJSON(400, "Failed to delete the tag")
        return
    }

    // Check if any rows were affected
    rowsAffected, _ := result.RowsAffected()
    if rowsAffected == 0 {
		txnLogger.Error().Str("tagId",tagId).Msg(`Tag not found`)
        ctx.AbortWithStatusJSON(404, "Tag not found")
        return
    }
	txnLogger.Info().Str("tagId",tagId).Msg(`Tag successfully deleted`)
    ctx.JSON(http.StatusOK, "User Removed")
}
func isUserExpired(tagId string) bool {
	var expireTime time.Time
	err := database.Db.QueryRow("select expire from tags where tag=$1", tagId).Scan(&expireTime)
	if err != nil {
		return false
	}
	logger.Debug().Msg("Expire time: " + expireTime.String())
	logger.Debug().Msg("Current time: " + time.Now().String())
	logger.Debug().Msg("Expire Difference" + time.Now().Sub(expireTime).String())
	return time.Now().After(expireTime)
}