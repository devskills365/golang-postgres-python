package handlers

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetRegions(db *pgxpool.Pool) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := db.Query(ctx, "SELECT nom_region FROM region WHERE nom_region IS NOT NULL")
	if err != nil {
		log.Printf("Erreur lors de la récupération des régions : %v", err)
		return nil, err
	}
	defer rows.Close()

	var regions []string
	for rows.Next() {
		var nomRegion string
		if err := rows.Scan(&nomRegion); err != nil {
			log.Printf("Erreur lors de la lecture d'une région : %v", err)
			return nil, err
		}
		log.Printf("Région : %s, octets : %v", nomRegion, []byte(nomRegion))
		regions = append(regions, nomRegion)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Erreur lors de l'itération des régions : %v", err)
		return nil, err
	}

	log.Printf("Régions récupérées depuis la base de données : %v", regions)
	return regions, nil
}
