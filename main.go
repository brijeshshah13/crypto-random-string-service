package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/brijeshshah13/crypto-random-string"
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"os"

	"github.com/cenkalti/backoff/v4"
	"github.com/cockroachdb/cockroach-go/v2/crdb"
)

func main() {
	r := gin.Default()

	db, err := initStore()
	if err != nil {
		log.Fatalf("failed to initialise the store: %s", err)
	}
	defer db.Close()

	r.GET("/", func(ctx *gin.Context) {
		rootHandler(ctx, db)
	})

	r.GET("/ping", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/random-string", func(ctx *gin.Context) {
		// just a practice on how to capture canceled requests
		select {
		case <-ctx.Request.Context().Done():
			fmt.Println(ctx.Request.Context().Err())
			return
		default:
		}
		generator := cryptorandomstring.New()
		if str, err := generator.WithLength(10).WithKind("hex").WithCharacters("abc").Generate(); err != nil {
			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"err": err.Error(),
			})
			return
		} else {
			ctx.JSON(http.StatusOK, gin.H{
				"string": str,
			})
		}
	})

	r.POST("/send", func(ctx *gin.Context) {
		sendHandler(ctx, db)
	})

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}

	r.Run(":" + httpPort)
}

type Message struct {
	Value string `json:"value"`
}

func initStore() (*sql.DB, error) {

	pgConnString := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=disable",
		os.Getenv("PGHOST"),
		os.Getenv("PGPORT"),
		os.Getenv("PGDATABASE"),
		os.Getenv("PGUSER"),
		os.Getenv("PGPASSWORD"),
	)

	var (
		db  *sql.DB
		err error
	)
	openDB := func() error {
		db, err = sql.Open("postgres", pgConnString)
		return err
	}

	err = backoff.Retry(openDB, backoff.NewExponentialBackOff())
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(
		"CREATE TABLE IF NOT EXISTS message (value STRING PRIMARY KEY)"); err != nil {
		return nil, err
	}

	return db, nil
}

func rootHandler(ctx *gin.Context, db *sql.DB) {
	r, err := countRecords(db)
	if err != nil {
		ctx.String(http.StatusInternalServerError, "%s", err.Error())
		return
	}
	ctx.String(http.StatusOK, "Hello, Docker! (%d)\n", r)
}

func countRecords(db *sql.DB) (int, error) {

	rows, err := db.Query("SELECT COUNT(*) FROM message")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, err
		}
		rows.Close()
	}

	return count, nil
}

func sendHandler(ctx *gin.Context, db *sql.DB) {

	m := &Message{}

	if err := ctx.Bind(m); err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"err": err.Error(),
		})
		return
	}

	err := crdb.ExecuteTx(context.Background(), db, nil,
		func(tx *sql.Tx) error {
			_, err := tx.Exec(
				"INSERT INTO message (value) VALUES ($1) ON CONFLICT (value) DO UPDATE SET value = excluded.value",
				m.Value,
			)
			if err != nil {
				return ctx.AbortWithError(http.StatusInternalServerError, err)
			}
			return nil
		})

	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"err": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, m)
}
