package speciessqlite

import (
	"testing"
	"time"
	"uuid"
	"workshop/internal/domain/species"
	"workshop/internal/infra/db/sqlc"

	"github.com/stretchr/testify/assert"
)

func TestTransform(t *testing.T) {
	t.Parallel()

	t.Run("Maps all fields", func(t *testing.T) {
		t.Parallel()

		id := uuid.New()
		createdAt := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
		updatedAt := createdAt.Add(time.Hour)
		input := db.Species{
			ID:             id,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			GbifKey:        5219404,
			Kingdom:        "Animalia",
			Phylum:         "Chordata",
			Class:          "Mammalia",
			Order:          "Carnivora",
			Family:         "Felidae",
			Genus:          "Panthera",
			Species:        "Panthera leo",
			Rank:           "SPECIES",
			ScientificName: "Panthera leo (Linnaeus, 1758)",
			CanonicalName:  "Panthera leo",
			VernacularName: "lion",
		}
		want := species.Species{
			ID:             id,
			GBIFKey:        5219404,
			Kingdom:        "Animalia",
			Phylum:         "Chordata",
			Class:          "Mammalia",
			Order:          "Carnivora",
			Family:         "Felidae",
			Genus:          "Panthera",
			Species:        "Panthera leo",
			Rank:           "SPECIES",
			ScientificName: "Panthera leo (Linnaeus, 1758)",
			CanonicalName:  "Panthera leo",
			VernacularName: "lion",
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
		}

		assert.Equal(t, want, transform(input))
	})

	t.Run("Preserves zero values", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, species.Species{}, transform(db.Species{}))
	})
}

func TestToSpeciesCreateParams(t *testing.T) {
	t.Parallel()

	t.Run("Maps all fields", func(t *testing.T) {
		t.Parallel()

		id := uuid.New()
		input := species.CreateParams{
			GBIFKey:        5219404,
			Kingdom:        "Animalia",
			Phylum:         "Chordata",
			Class:          "Mammalia",
			Order:          "Carnivora",
			Family:         "Felidae",
			Genus:          "Panthera",
			Species:        "Panthera leo",
			Rank:           "SPECIES",
			ScientificName: "Panthera leo (Linnaeus, 1758)",
			CanonicalName:  "Panthera leo",
			VernacularName: "lion",
		}
		want := db.SpeciesCreateParams{
			ID:             id,
			GbifKey:        5219404,
			Kingdom:        "Animalia",
			Phylum:         "Chordata",
			Class:          "Mammalia",
			Order:          "Carnivora",
			Family:         "Felidae",
			Genus:          "Panthera",
			Species:        "Panthera leo",
			Rank:           "SPECIES",
			ScientificName: "Panthera leo (Linnaeus, 1758)",
			CanonicalName:  "Panthera leo",
			VernacularName: "lion",
		}

		assert.Equal(t, want, toSpeciesCreateParams(id, input))
	})

	t.Run("Preserves zero values", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, db.SpeciesCreateParams{}, toSpeciesCreateParams(uuid.Nil(), species.CreateParams{}))
	})
}
