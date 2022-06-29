package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v4"
)

func main() {
	cb := NewContextBrokerClient("http://localhost:8082")
	ctx := context.Background()
	beaches := cb.GetBeaches(ctx)

	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	for _, b := range beaches {
		if temp, ok := b.GetLatestTemperature(); ok {
			err = conn.BeginFunc(ctx, func(tx pgx.Tx) error {
				update := fmt.Sprintf("update geodata_cip.beaches set temperature=%g, timestampObservered='%s', temperatureSource='%s' where serviceGuideId='%s'", temp.Value, temp.DateObserved.Format(time.RFC3339), temp.Source, b.Source)
				_, err = tx.Exec(ctx, update)
				return err
			})
		}
	}
}

/*
CREATE TABLE geodata_cip.beaches
(
"id" integer NOT NULL,
"serviceGuideId" text COLLATE pg_catalog."default",
"name" text COLLATE pg_catalog."default",
"serviceTypes" text COLLATE pg_catalog."default",
"webPage" text COLLATE pg_catalog."default",
"visitingAddress" text COLLATE pg_catalog."default",
"temperature" numeric,
"timestampObservered" timestamp,
"temperatureSource" text COLLATE pg_catalog."default",
"geom" geometry(Geometry,3007),
CONSTRAINT beaches_pkey PRIMARY KEY ("id")
)
*/