package weathersqlite

import (
	"testing"
	"time"
	"uuid"
	"workshop/internal/domain/weather"
	"workshop/internal/infra/db/sqlc"

	"github.com/stretchr/testify/assert"
)

func TestTransform(t *testing.T) {
	t.Parallel()

	t.Run("Maps all fields", func(t *testing.T) {
		t.Parallel()

		id := uuid.New()
		observedAt := time.Date(2026, 9, 22, 14, 0, 0, 0, time.UTC)
		createdAt := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
		updatedAt := createdAt.Add(time.Hour)
		input := db.Weather{
			ID:                  id,
			ObservedAt:          observedAt,
			Temperature:         24.2,
			ApparentTemperature: 26.0,
			Humidity:            64,
			Precipitation:       2.6,
			Rain:                2.4,
			Snowfall:            0.1,
			CloudCover:          72,
			WindSpeed:           21.3,
			WindGusts:           38.9,
			WindDirection:       135,
			ConditionCode:       61,
			Condition:           "Slight rain",
			CreatedAt:           createdAt,
			UpdatedAt:           updatedAt,
		}
		want := weather.Weather{
			ID:            id,
			ObservedAt:    observedAt,
			Temperature:   weather.Temperature{Actual: 24.2, Apparent: 26.0},
			Precipitation: weather.Precipitation{Total: 2.6, Rain: 2.4, Snowfall: 0.1},
			Wind:          weather.Wind{Speed: 21.3, Gusts: 38.9, Direction: 135},
			Condition:     weather.Condition{Code: 61, Description: "Slight rain"},
			CloudCover:    72,
			Humidity:      64,
			CreatedAt:     createdAt,
			UpdatedAt:     updatedAt,
		}

		assert.Equal(t, want, transform(input))
	})

	t.Run("Preserves zero values", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, weather.Weather{}, transform(db.Weather{}))
	})
}

func TestToWeatherCreateParams(t *testing.T) {
	t.Parallel()

	t.Run("Maps all fields", func(t *testing.T) {
		t.Parallel()

		id := uuid.New()
		observedAt := time.Date(2026, 9, 22, 14, 0, 0, 0, time.UTC)
		input := weather.CreateParams{
			ObservedAt:    observedAt,
			Temperature:   weather.Temperature{Actual: 24.2, Apparent: 26.0},
			Precipitation: weather.Precipitation{Total: 2.6, Rain: 2.4, Snowfall: 0.1},
			Wind:          weather.Wind{Speed: 21.3, Gusts: 38.9, Direction: 135},
			Condition:     weather.Condition{Code: 61, Description: "Slight rain"},
			CloudCover:    72,
			Humidity:      64,
		}
		want := db.WeatherCreateParams{
			ID:                  id,
			ObservedAt:          observedAt,
			Temperature:         24.2,
			ApparentTemperature: 26.0,
			Humidity:            64,
			Precipitation:       2.6,
			Rain:                2.4,
			Snowfall:            0.1,
			CloudCover:          72,
			WindSpeed:           21.3,
			WindGusts:           38.9,
			WindDirection:       135,
			ConditionCode:       61,
			Condition:           "Slight rain",
		}

		assert.Equal(t, want, toWeatherCreateParams(id, input))
	})

	t.Run("Preserves zero values", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, db.WeatherCreateParams{}, toWeatherCreateParams(uuid.Nil(), weather.CreateParams{}))
	})
}
