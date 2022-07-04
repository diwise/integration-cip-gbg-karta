package main

import (
	"context"
	"fmt"
	"time"

	"github.com/diwise/integration-cip-gbg-karta/internal/pkg/application"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/jackc/pgx/v4"
)

const serviceName string = "integration-cip-gbg-karta"

func main() {
	serviceVersion := buildinfo.SourceVersion()

	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	contextBrokerUrl := env.GetVariableOrDie(logger, "CONTEXT_BROKER_URL", "url to context broker")
	pgConnUrl := env.GetVariableOrDie(logger, "PG_CONNECTION_URL", "url to postgres database, i.e. postgres://username:password@hostname:5433/database_name")

	cb := application.NewContextBrokerClient(contextBrokerUrl)

	beaches, err := cb.GetBeaches(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to fetch beaches")
	}

	logger.Info().Msgf("fetched %d beaches from %s", len(beaches), contextBrokerUrl)

	conn, err := pgx.Connect(ctx, pgConnUrl)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to connect to database")
	}
	defer conn.Close(ctx)

	for _, b := range beaches {
		if temp, ok := b.GetLatestTemperature(ctx); ok {
			err = conn.BeginFunc(ctx, func(tx pgx.Tx) error {
				//dateStr, timeStr := ToSwedishDateAndTime(temp.DateObserved)
				update := fmt.Sprintf("update geodata_cip.beaches set \"temperature\"='%0.1f', \"timestampObservered\"='%s', \"temperatureSource\"='%s' where \"serviceGuideId\"='%s'", temp.Value, temp.DateObserved.Format(time.RFC3339), temp.Source, b.Source)
				_, err = tx.Exec(ctx, update)
				if err != nil {
					return err
				}

				logger.Info().Msgf("updated temperature value for %s (%s)", b.Name, b.Source)

				return nil
			})
			if err != nil {
				logger.Error().Err(err).Msg("unable to update table")
			}
		} else {
			logger.Warn().Msgf("no valid temperature value found for %s (%s)", b.Name, b.Source)
		}
	}

	logger.Info().Msg("done")
}

var months []string = []string{
	"januari", "februari", "mars", "april", "maj", "juni",
	"juli", "augusti", "september", "oktober", "november", "december",
}

func ToSwedishDateAndTime(t time.Time) (string, string) {
	cest := t.Add(2 * time.Hour)

	dateStr := fmt.Sprintf("%d %s %d", cest.Day(), months[cest.Month()-1], cest.Year())

	return dateStr, cest.Format("15.04")
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
