package main

import (
	"testing"
	"time"

	"github.com/matryer/is"
)

func TestSwedishDateFormatting(t *testing.T) {
	is := is.New(t)

	theTime, err := time.Parse(time.RFC3339, "2022-07-03T12:14:15Z")
	is.NoErr(err)

	dateStr, _ := ToSwedishDateAndTime(theTime)

	is.Equal(dateStr, "3 juli 2022")
}

func TestSwedishTimeFormatting(t *testing.T) {
	is := is.New(t)

	theTime, err := time.Parse(time.RFC3339, "2022-07-03T12:14:15Z")
	is.NoErr(err)

	_, timeStr := ToSwedishDateAndTime(theTime)

	is.Equal(timeStr, "14.14")
}
