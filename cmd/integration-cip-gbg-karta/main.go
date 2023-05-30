package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
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

	getWebSite := func(strs []string) string {
		for _, s := range strs {
			if strings.HasPrefix(s, "https://goteborg.se/") {
				return s
			}
		}
		return ""
	}

	err = createBeachTableIfNotExists(ctx, conn)
	if err != nil {
		return err
	}

	errs := []error{}

	for _, b := range beaches {
		log := logger.With().Str("source", b.Source).Logger().With().Str("name", b.Name).Logger()

		var n int
		err = conn.QueryRow(ctx, "select count(*) as \"n\" from geodata_cip.beaches where \"serviceGuideId\"=$1", b.Source).Scan(&n)
		if err != nil {
			errs = append(errs, fmt.Errorf("could not count, %w", err))
			continue
		}

		if n == 0 {
			lat, lon := b.AsPoint()
			_, err = conn.Exec(ctx, "insert into geodata_cip.beaches(\"serviceGuideId\",\"name\",\"serviceTypes\",\"webPage\",\"geom\") values($1,$2,$3,$4,ST_MakePoint($5,$6))", b.Source, b.Name, strings.Join(b.BeachType, ", "), getWebSite(b.SeeAlso), lat, lon)
			if err != nil {
				errs = append(errs, fmt.Errorf("could not insert new beach, %w", err))
				continue
			}

			log.Debug().Msg("new beach inserted")
		}

		if temp, ok := models.CalcLastTemperatureObserved(b); ok {
			_, err = conn.Exec(ctx, "update geodata_cip.beaches set \"temperature\"=$1, \"timestampObservered\"=$2, \"temperatureSource\"=$3, \"name\"=$5, \"serviceTypes\"=$6, \"webPage\"=$7  where \"serviceGuideId\"=$4", temp.Value, temp.DateObserved.Format(time.RFC3339), temp.Source, b.Source, b.Name, strings.Join(b.BeachType, ", "), getWebSite(b.SeeAlso))
			if err != nil {
				errs = append(errs, fmt.Errorf("could not update temperature, %w", err))
				continue
			}

			log.Debug().Msg("temperature updated")
		} else {
			_, err = conn.Exec(ctx, "update geodata_cip.beaches set \"temperature\"=null, \"timestampObservered\"=null, \"temperatureSource\"=null where \"serviceGuideId\"=$1", b.Source)
			if err != nil {
				errs = append(errs, fmt.Errorf("could not set temperatures to NULL, %w", err))
				continue
			}

			log.Debug().Msgf("cleared temperatures since no valid temperatures was found")
		}
	}

	return errors.Join(errs...)
}

func createBeachTableIfNotExists(ctx context.Context, conn pgx.Conn) error {
	_, err := conn.Exec(ctx, `
		CREATE SCHEMA IF NOT EXISTS geodata_cip;
		
		CREATE TABLE IF NOT EXISTS geodata_cip.beaches
		(
			"id" SERIAL,
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
		);

		CREATE UNIQUE INDEX IF NOT EXISTS beaches_sgid_idx ON geodata_cip.beaches ("serviceGuideId");`)
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
