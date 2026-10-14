package species

import (
	"time"
	"uuid"
)

type (
	// Species represents a taxonomy record as GBIF classifies it.
	// It can be an animal or a plant, at species rank or above.
	Species struct {
		ID             uuid.UUID `json:"id"`
		GBIFKey        int       `json:"gbifKey"`
		Kingdom        string    `json:"kingdom,omitzero"`
		Phylum         string    `json:"phylum,omitzero"`
		Class          string    `json:"class,omitzero"`
		Order          string    `json:"order,omitzero"`
		Family         string    `json:"family,omitzero"`
		Genus          string    `json:"genus,omitzero"`
		Species        string    `json:"species,omitzero"`
		Rank           string    `json:"rank,omitzero"`
		ScientificName string    `json:"scientificName,omitzero"`
		CanonicalName  string    `json:"canonicalName,omitzero"`
		VernacularName string    `json:"vernacularName,omitzero"`
		CreatedAt      time.Time `json:"createdAt"`
		UpdatedAt      time.Time `json:"updatedAt"`
	}
	// CreateParams represents the parameters required to
	// create a new species.
	CreateParams struct {
		GBIFKey        int    `json:"gbifKey"`
		Kingdom        string `json:"kingdom,omitzero"`
		Phylum         string `json:"phylum,omitzero"`
		Class          string `json:"class,omitzero"`
		Order          string `json:"order,omitzero"`
		Family         string `json:"family,omitzero"`
		Genus          string `json:"genus,omitzero"`
		Species        string `json:"species,omitzero"`
		Rank           string `json:"rank,omitzero"`
		ScientificName string `json:"scientificName,omitzero"`
		CanonicalName  string `json:"canonicalName,omitzero"`
		VernacularName string `json:"vernacularName,omitzero"`
	}
)

// CreateParams returns the species fields without its ID or timestamps.
func (s Species) CreateParams() CreateParams {
	return CreateParams{
		GBIFKey:        s.GBIFKey,
		Kingdom:        s.Kingdom,
		Phylum:         s.Phylum,
		Class:          s.Class,
		Order:          s.Order,
		Family:         s.Family,
		Genus:          s.Genus,
		Species:        s.Species,
		Rank:           s.Rank,
		ScientificName: s.ScientificName,
		CanonicalName:  s.CanonicalName,
		VernacularName: s.VernacularName,
	}
}
