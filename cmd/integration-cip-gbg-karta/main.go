package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/diwise/integration-cip-gbg-karta/internal/pkg/application"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog"
)

const serviceName string = "integration-cip-gbg-karta"

var bcSelector string

func main() {
	serviceVersion := buildinfo.SourceVersion()

	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	contextBrokerUrl := env.GetVariableOrDie(logger, "CONTEXT_BROKER_URL", "url to context broker")
	pgConnUrl := env.GetVariableOrDie(logger, "PG_CONNECTION_URL", "url to postgres database, i.e. postgres://username:password@hostname:5433/database_name")

	cb := application.NewContextBrokerClient(contextBrokerUrl)

	flag.StringVar(&bcSelector, "bcSelector", "beach", "Flag to distinguish which funcion to be selected for its business case")
	conn, err := pgx.Connect(ctx, pgConnUrl)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to connect to database")
	}

	defer conn.Close(ctx)

	if bcSelector == "beach" {
		err := bcWaterQualityObserved(ctx, cb, logger, contextBrokerUrl, *conn)
		if err != nil {
			logger.Error().Err(err).Msg("error in bcWaterQualityObserved")
		}
	} else if bcSelector == "greenspacerecord" {
		err := bcGreenspaceRecord(ctx, cb, logger, contextBrokerUrl, *conn)
		if err != nil {
			logger.Error().Err(err).Msg("error in bcGreenspaceRecord")
		}
	} else {
		logger.Fatal().Err(err).Msgf("%s is not a supported business case", bcSelector)
	}

	logger.Info().Msg("done")
}

func bcWaterQualityObserved(ctx context.Context, cb application.ContextBrokerClient, logger zerolog.Logger, contextBrokerUrl string, conn pgx.Conn) error {
	beaches, err := cb.GetBeaches(ctx)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to fetch beaches")
	}

	logger.Info().Msgf("fetched %d beaches from %s", len(beaches), contextBrokerUrl)

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

	return err
}

func bcGreenspaceRecord(ctx context.Context, cb application.ContextBrokerClient, logger zerolog.Logger, contextBrokerUrl string, conn pgx.Conn) error {
	greenspaces, err := cb.GetGreenspaceRecords(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("no greenspacerecords fetched")
		return nil
	}

	logger.Info().Msgf("fetched %d greenspacerecords from %s", len(greenspaces), contextBrokerUrl)

	for _, g := range greenspaces {
		err = conn.BeginFunc(ctx, func(tx pgx.Tx) error {
			lon := g.Location.Coordinates[0]
			lat := g.Location.Coordinates[1]
			update := fmt.Sprintf("insert into geodata_markfukt.greenspacerecord (\"id\", \"location\", \"soilMoisturePressure\", \"dateObservered\", \"source\") VALUES ('%s',  ST_MakePoint(%f,%f), %d, '%s', '%s')", g.Id, lon, lat, g.SoilMoisturePressure, g.DateObserved.Value, "Göteborg stads park- och naturnämnd")

			_, err = tx.Exec(ctx, update)
			if err != nil {
				return err
			}

			logger.Info().Msg("updated soilmoisture pressure into geodata_markfukt.greenspacerecord")

			return nil
		})
		if err != nil {
			logger.Error().Err(err).Msg("unable to update table")
		}
	}
	return err

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
