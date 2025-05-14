package handlers

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DataEntry represents a single row from the database query
type DataEntry struct {
	Domaine     string
	Sousdomaine string
	Indicateur  string
}

// GetData fetches domains, subdomains, and indicators and structures them as a map
func GetData(db *pgxpool.Pool) (map[string][]string, error) {
	query := `
		SELECT d.domaine, sd.sousdomaine, i.indicateur
		FROM domaines d
		JOIN sousdomaines sd ON d.domaine_id = sd.domaines_id
		JOIN indicateurs i ON sd.sousdomaine_id = i.sousdomaine_id
		ORDER BY d.domaine, sd.sousdomaine, i.indicateur;
	`

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		log.Printf("Error executing query: %v", err)
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Structure: map["domaine, sousdomaine"][]indicateur
	data := make(map[string][]string)
	for rows.Next() {
		var entry DataEntry
		if err := rows.Scan(&entry.Domaine, &entry.Sousdomaine, &entry.Indicateur); err != nil {
			log.Printf("Error scanning row: %v", err)
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		key := fmt.Sprintf("%s, %s", entry.Domaine, entry.Sousdomaine)
		data[key] = append(data[key], entry.Indicateur)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating rows: %v", err)
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return data, nil
}
