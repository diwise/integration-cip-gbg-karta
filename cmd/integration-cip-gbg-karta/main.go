package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/diwise/integration-cip-gbg-karta/internal/pkg/application"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/jackc/pgx/v4"
)

func main() {
	serviceName := "integration-cip-gbg-karta"
	serviceVersion := buildinfo.SourceVersion()

	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	contextBrokerUrl := env.GetVariableOrDie(logger, "CONTEXT_BROKER_URL", "url to context broker")
	pgConnUrl := env.GetVariableOrDie(logger, "PG_CONNECTION_URL", "url to postgres database, i.e. postgres://username:password@hostname:5433/database_name")

	cb := application.NewContextBrokerClient(contextBrokerUrl)

	beaches, err := cb.GetBeaches(ctx)
	if err != nil {
		logger.Err(err).Msg("unable to fetch beaches")
		os.Exit(1)
	}

	conn, err := pgx.Connect(ctx, pgConnUrl)
	if err != nil {
		logger.Err(err).Msg("unable to connect to database")
		os.Exit(1)
	}
	defer conn.Close(ctx)

	for _, b := range beaches {
		if temp, ok := b.GetLatestTemperature(); ok {
			err = conn.BeginFunc(ctx, func(tx pgx.Tx) error {
				update := fmt.Sprintf("update geodata_cip.beaches set \"temperature\"='%g', \"timestampObservered\"='%s', \"temperatureSource\"='%s' where \"serviceGuideId\"='%s'", temp.Value, temp.DateObserved.Format(time.RFC3339), temp.Source, b.Source)
				_, err = tx.Exec(ctx, update)
				return err
			})
			if err != nil {
				logger.Err(err).Msg("unable to update table")
			}
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
