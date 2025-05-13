package handlers

import (
	"context"
	"log"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgxpool"
)

// GetIndicateurs récupère la liste des indicateurs distincts non nuls
// Les chaînes de caractères sont encodées en UTF-8 pour gérer les caractères spéciaux
func GetIndicateurs(db *pgxpool.Pool) ([]string, error) {
	query := `SELECT DISTINCT indicateur FROM indicateurs WHERE indicateur IS NOT NULL`

	rows, err := db.Query(context.Background(), query)
	if err != nil {
		log.Printf("Erreur lors de la récupération des indicateurs : %v", err)
		return nil, err
	}
	defer rows.Close()

	var indicateurs []string
	for rows.Next() {
		var indicateur string
		if err := rows.Scan(&indicateur); err != nil {
			log.Printf("Erreur lors de la lecture d'un indicateur : %v", err)
			return nil, err
		}
		// Vérifier l'encodage UTF-8
		if !utf8.ValidString(indicateur) {
			log.Printf("Indicateur non valide en UTF-8: %s, octets: %v", indicateur, []byte(indicateur))
		} else {
			log.Printf("Indicateur: %s, octets: %v", indicateur, []byte(indicateur))
		}
		indicateurs = append(indicateurs, indicateur)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Erreur lors de l'itération des résultats : %v", err)
		return nil, err
	}

	return indicateurs, nil
}
