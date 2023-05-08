package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/diwise/integration-cip-gbg-karta/internal/pkg/cip"
	"github.com/diwise/integration-cip-gbg-karta/internal/pkg/models"
	"github.com/diwise/service-chassis/pkg/infrastructure/buildinfo"
	"github.com/diwise/service-chassis/pkg/infrastructure/env"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
	"github.com/jackc/pgx/v4"
)

const serviceName string = "integration-cip-gbg-karta"

var bcSelector string
var maxDistance string

func main() {
	var err error

	serviceVersion := buildinfo.SourceVersion()

	ctx, logger, cleanup := o11y.Init(context.Background(), serviceName, serviceVersion)
	defer cleanup()

	contextBrokerUrl := env.GetVariableOrDie(logger, "CONTEXT_BROKER_URL", "url to context broker")
	pgConnUrl := env.GetVariableOrDie(logger, "PG_CONNECTION_URL", "url to postgres database, i.e. postgres://username:password@hostname:5433/database_name")

	flag.StringVar(&bcSelector, "bc", "beach", "selected business case [beach or greenspacerecord]")
	flag.StringVar(&maxDistance, "distance", "500", "max distance between beach and temperature measurement, default 500m")
	flag.Parse()

	conn, err := pgx.Connect(ctx, pgConnUrl)
	if err != nil {
		logger.Fatal().Err(err).Msg("unable to connect to database")
	}
	defer conn.Close(ctx)

	switch bcSelector {
	case "beach":
		err = bcWaterQualityObserved(ctx, contextBrokerUrl, maxDistance, *conn)
	case "greenspacerecord":
		err = bcGreenspaceRecord(ctx, contextBrokerUrl, *conn)
	default:
		logger.Fatal().Msgf("%s is not a supported business case", bcSelector)
	}

	if err != nil {
		logger.Error().Err(err).Msgf("error occured when running integration")
	}

	logger.Info().Msg("done")
}

func bcWaterQualityObserved(ctx context.Context, contextBrokerUrl, maxDistance string, conn pgx.Conn) error {
	logger := logging.GetFromContext(ctx)

	beaches, err := cip.GetBeachesWithTemp(ctx, contextBrokerUrl, maxDistance)
	if err != nil {
		return err
	}

	for i, b := range beaches {
		if temp, ok := models.CalcLastTemperatureObserved(b); ok {
			err = conn.BeginFunc(ctx, func(tx pgx.Tx) error {
				var n int
				err = conn.QueryRow(ctx, "select count(*) as \"n\" from geodata_cip.beaches where \"serviceGuideId\"=$1", b.Source).Scan(&n)
				if err != nil {
					return fmt.Errorf("could not count, %w", err)
				}

				if n == 0 {
					lat, lon := b.AsPoint()
					_, err = tx.Exec(ctx, "insert into geodata_cip.beaches(\"id\",\"serviceGuideId\",\"name\",\"serviceTypes\",\"webPage\",\"visitingAddress\",\"temperature\",\"timestampObservered\",\"temperatureSource\",\"geom\") values($1,$2,$3,$4,$5,$6,$7,$8,$9,ST_MakePoint($10,$11))", i, b.Source, b.Name, "", "", "", temp.Value, temp.DateObserved.Format(time.RFC3339), temp.Source, lat, lon)					
					if err != nil {
						return fmt.Errorf("could not insert, %w", err)
					}
					logger.Debug().Msgf("added temperature value for %s (%s)", b.Name, b.Source)
					return nil
				}

				_, err = tx.Exec(ctx, "update geodata_cip.beaches set \"temperature\"=$1, \"timestampObservered\"=$2, \"temperatureSource\"=$3 where \"serviceGuideId\"=$4", temp.Value, temp.DateObserved.Format(time.RFC3339), temp.Source, b.Source)
				if err != nil {
					return fmt.Errorf("could not update, %w", err)
				}

				logger.Debug().Msgf("updated temperature value for %s (%s)", b.Name, b.Source)

				return nil
			})
			if err != nil {
				logger.Error().Err(err).Msg("faild to add or update data")
			}
		} else {
			logger.Debug().Msgf("no valid temperature value found for %s (%s)", b.Name, b.Source)
		}
	}

	return err
}

func bcGreenspaceRecord(ctx context.Context, contextBrokerUrl string, conn pgx.Conn) error {
	logger := logging.GetFromContext(ctx)

	greenspaces, err := cip.GetGreenspaceRecords(ctx, contextBrokerUrl)
	if err != nil {
		logger.Error().Err(err).Msg("no greenspacerecords fetched")
		return nil
	}

	logger.Info().Msgf("fetched %d greenspacerecords from %s", len(greenspaces), contextBrokerUrl)

	for _, g := range greenspaces {
		err = conn.BeginFunc(ctx, func(tx pgx.Tx) error {
			lon := g.Location.Coordinates[0]
			lat := g.Location.Coordinates[1]
			insert := fmt.Sprintf("insert into geodata_markfukt.greenspacerecord (\"id\", \"location\", \"soilMoisturePressure\", \"dateObservered\", \"source\") VALUES ('%s',  ST_MakePoint(%f,%f), %d, '%s', '%s') ON CONFLICT DO NOTHING;", g.Id, lon, lat, g.SoilMoisturePressure, g.DateObserved.Value, "Göteborg stads park- och naturnämnd")

			_, err = tx.Exec(ctx, insert)
			if err != nil {
				return err
			}

			logger.Info().Msg("inserted soilmoisture pressure into geodata_markfukt.greenspacerecord")

			return nil
		})
		if err != nil {
			logger.Error().Err(err).Msg("unable to insert into table")
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
