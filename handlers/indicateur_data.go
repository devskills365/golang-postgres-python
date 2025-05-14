package handlers

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// IndicateurData represents a row from datav0
type IndicateurData struct {
	Indicateur string  `json:"indicateurs"`
	Dimension  string  `json:"dimension"`
	Modalites  string  `json:"modalites"`
	Valeur     float64 `json:"valeurs"`
	Annee      string  `json:"annee"`
	ClePivot   string  `json:"cle_pivot_table"`
	Definition string  `json:"Definition"`
	ModeCalcul string  `json:"ModeCalcul"`
}

// ProcessedData represents a processed row with dimensions as key-value pairs
type ProcessedData struct {
	Indicateur string            `json:"indicateurs"`
	Valeur     float64           `json:"valeurs"`
	Annee      string            `json:"annee"`
	ClePivot   string            `json:"cle_pivot_table"`
	Dimensions map[string]string `json:"Dimensions"`
}

// PivotRequest represents the AJAX request for pivot table
type PivotRequest struct {
	RowColumns  []string `json:"row_columns"`
	ColColumns  []string `json:"col_columns"`
	ValueColumn string   `json:"value_column"`
}

// PivotResponse represents the pivot table response
type PivotResponse struct {
	Columns [][]string      `json:"columns"`
	Index   []string        `json:"index"`
	Data    [][]interface{} `json:"data"`
}

// GetIndicateurData fetches data for a specific indicator from datav0

// GetIndicateurData récupère les données pour un indicateur spécifique depuis datav0
func GetIndicateurData(db *pgxpool.Pool, indicateur string) ([]IndicateurData, error) {
	// Nettoyer l'indicateur en entrée pour gérer les espaces ou apostrophes
	indicateur = strings.TrimSpace(indicateur)
	log.Printf("Requête pour l'indicateur : %q", indicateur)

	query := `
		SELECT 
			d.indicateurs,
			d.dimension,
			d.modalites,
			d.valeurs,
			d.annee::TEXT,
			COALESCE(i.definitions, '') AS definition,
			COALESCE(i.mode_calcul, '') AS mode_calcul
		FROM datav0 d
		LEFT JOIN indicateurs i ON LOWER(d.indicateurs) = LOWER(i.indicateur)
		WHERE LOWER(d.indicateurs) = LOWER($1)
		ORDER BY d.indicateurs, d.annee, d.dimension;
	`
	rows, err := db.Query(context.Background(), query, indicateur)
	if err != nil {
		log.Printf("Erreur lors de la requête sur datav0 : %v", err)
		return nil, fmt.Errorf("échec de la requête sur datav0 : %w", err)
	}
	defer rows.Close()

	var data []IndicateurData
	for rows.Next() {
		var d IndicateurData
		if err := rows.Scan(&d.Indicateur, &d.Dimension, &d.Modalites, &d.Valeur, &d.Annee, &d.Definition, &d.ModeCalcul); err != nil {
			log.Printf("Erreur lors de la lecture de la ligne : %v", err)
			return nil, fmt.Errorf("échec de la lecture de la ligne : %w", err)
		}
		// Nettoyer les chaînes
		d.Indicateur = strings.TrimSpace(d.Indicateur)
		d.Dimension = strings.TrimSpace(d.Dimension)
		d.Modalites = strings.TrimSpace(d.Modalites)
		d.Annee = strings.TrimSpace(d.Annee)
		d.Definition = strings.TrimSpace(d.Definition)
		d.ModeCalcul = strings.TrimSpace(d.ModeCalcul)
		// Construire cle_pivot_table
		if d.Dimension != "" {
			dims := strings.Split(d.Dimension, " / ")
			for i, dim := range dims {
				dims[i] = strings.TrimSpace(dim)
			}
			d.ClePivot = strings.Join(dims, ",") + ",annee"
		}
		data = append(data, d)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Erreur lors de l'itération des lignes : %v", err)
		return nil, fmt.Errorf("erreur lors de l'itération des lignes : %w", err)
	}

	log.Printf("Nombre de lignes récupérées pour l'indicateur %q : %d", indicateur, len(data))
	return data, nil
}

// ProcessIndicateurData transforms raw data into processed format

// ProcessIndicateurData transforme les données brutes en format traité
func ProcessIndicateurData(rawData []IndicateurData) ([]ProcessedData, []string) {
	var processed []ProcessedData
	var desaggregationColumns []string
	seenColumns := make(map[string]bool)
	maxRows := 50000 // Limite à 1000 lignes
	rowCount := 0

	for _, row := range rawData {
		// Vérifier si la limite de lignes est atteinte
		if rowCount >= maxRows {
			log.Printf("Limite de %d lignes atteinte, arrêt du traitement pour l'indicateur %q", maxRows, row.Indicateur)
			break
		}

		// Valider les champs obligatoires
		if row.Indicateur == "" || row.Dimension == "" || row.Modalites == "" {
			log.Printf("Ignorer ligne invalide : indicateur=%q, dimension=%q, modalites=%q", row.Indicateur, row.Dimension, row.Modalites)
			continue
		}

		// Séparer dimensions et modalités
		dimCols := strings.Split(row.Dimension, " / ")
		modValues := strings.Split(row.Modalites, " / ")
		if len(dimCols) != len(modValues) {
			log.Printf("Incohérence entre dimensions (%d) et modalités (%d) pour l'indicateur : %s", len(dimCols), len(modValues), row.Indicateur)
			continue
		}

		// Nettoyer dimensions et modalités
		for i, col := range dimCols {
			dimCols[i] = strings.TrimSpace(col)
			if col != "" && !seenColumns[col] {
				desaggregationColumns = append(desaggregationColumns, col)
				seenColumns[col] = true
			}
		}
		for i, val := range modValues {
			modValues[i] = strings.TrimSpace(val)
		}

		// Créer la carte des dimensions
		dimMap := make(map[string]string)
		for i, col := range dimCols {
			if i < len(modValues) {
				dimMap[col] = modValues[i]
			}
		}

		// Construire la ligne traitée
		processedRow := ProcessedData{
			Indicateur: row.Indicateur,
			Valeur:     row.Valeur,
			Annee:      row.Annee,
			ClePivot:   row.ClePivot,
			Dimensions: dimMap,
		}
		processed = append(processed, processedRow)
		rowCount++ // Incrémenter le compteur après ajout
	}

	// Trier et nettoyer les colonnes de désagrégation
	for i, col := range desaggregationColumns {
		desaggregationColumns[i] = strings.TrimSpace(col)
	}

	log.Printf("Nombre de lignes traitées : %d (limite : %d)", len(processed), maxRows)
	return processed, desaggregationColumns
}

// CreatePivotTable generates a pivot table based on row/column selections
func CreatePivotTable(data []ProcessedData, req PivotRequest) (PivotResponse, error) {
	// Validate inputs
	if len(req.RowColumns) == 0 && len(req.ColColumns) == 0 {
		return PivotResponse{}, fmt.Errorf("at least one row or column must be specified")
	}
	if req.ValueColumn != "valeurs" {
		return PivotResponse{}, fmt.Errorf("only 'valeurs' is supported as value column")
	}

	// Combine row and column dimensions for filtering
	myIndex := append(req.RowColumns, req.ColColumns...)
	myIndexSet := make(map[string]bool)
	for _, col := range myIndex {
		myIndexSet[col] = true
	}

	// Filter data based on cle_pivot_table
	var filtered []ProcessedData
	for _, row := range data {
		if row.ClePivot == "" {
			continue
		}
		cleDims := strings.Split(row.ClePivot, ",")
		cleSet := make(map[string]bool)
		for _, dim := range cleDims {
			cleSet[strings.TrimSpace(dim)] = true
		}
		// Check if cle_pivot_table matches myIndex
		matches := true
		for dim := range myIndexSet {
			if !cleSet[dim] {
				matches = false
				break
			}
		}
		for dim := range cleSet {
			if !myIndexSet[dim] && dim != "annee" {
				matches = false
				break
			}
		}
		if matches {
			filtered = append(filtered, row)
		}
	}

	if len(filtered) == 0 {
		return PivotResponse{}, fmt.Errorf("no data matches the selected dimensions")
	}

	// Build pivot table
	type pivotKey struct {
		RowKey string
		ColKey string
	}
	pivotData := make(map[pivotKey]float64)
	rowKeys := make(map[string]bool)
	colKeys := make(map[string]bool)

	for _, row := range filtered {
		// Build row key
		var rowParts []string
		for _, rowCol := range req.RowColumns {
			if val, ok := row.Dimensions[rowCol]; ok {
				rowParts = append(rowParts, val)
			} else {
				rowParts = append(rowParts, "-")
			}
		}
		rowKey := strings.Join(rowParts, "|")
		if rowKey != "" {
			rowKeys[rowKey] = true
		}

		// Build column key
		var colParts []string
		for _, colCol := range req.ColColumns {
			if val, ok := row.Dimensions[colCol]; ok {
				colParts = append(colParts, val)
			} else {
				colParts = append(colParts, "-")
			}
		}
		colKey := strings.Join(colParts, "|")
		if colKey != "" {
			colKeys[colKey] = true
		}

		// Aggregate value
		key := pivotKey{RowKey: rowKey, ColKey: colKey}
		pivotData[key] += row.Valeur
	}

	// Prepare response
	var sortedRowKeys []string
	for k := range rowKeys {
		sortedRowKeys = append(sortedRowKeys, k)
	}
	var sortedColKeys []string
	for k := range colKeys {
		sortedColKeys = append(sortedColKeys, k)
	}
	if len(sortedRowKeys) == 0 {
		sortedRowKeys = append(sortedRowKeys, "")
	}
	if len(sortedColKeys) == 0 {
		sortedColKeys = append(sortedColKeys, "")
	}

	// Build columns
	var columns [][]string
	if len(req.ColColumns) > 0 {
		for _, colKey := range sortedColKeys {
			parts := strings.Split(colKey, "|")
			columns = append(columns, parts)
		}
	} else {
		columns = append(columns, []string{"valeurs"})
	}

	// Build data
	dataMatrix := make([][]interface{}, len(sortedRowKeys))
	for i, rowKey := range sortedRowKeys {
		dataMatrix[i] = make([]interface{}, len(sortedColKeys))
		for j, colKey := range sortedColKeys {
			val := pivotData[pivotKey{RowKey: rowKey, ColKey: colKey}]
			dataMatrix[i][j] = val
		}
	}

	return PivotResponse{
		Columns: columns,
		Index:   sortedRowKeys,
		Data:    dataMatrix,
	}, nil
}

// GetAllIndicateurs fetches all unique indicators from datav0
func GetAllIndicateurs(db *pgxpool.Pool) ([]string, error) {
	query := `SELECT DISTINCT indicateurs FROM datav0 ORDER BY indicateurs;`
	rows, err := db.Query(context.Background(), query)
	if err != nil {
		log.Printf("Error querying indicators: %v", err)
		return nil, fmt.Errorf("failed to query indicators: %w", err)
	}
	defer rows.Close()

	var indicateurs []string
	for rows.Next() {
		var ind string
		if err := rows.Scan(&ind); err != nil {
			log.Printf("Error scanning indicator: %v", err)
			return nil, fmt.Errorf("failed to scan indicator: %w", err)
		}
		ind = strings.TrimSpace(ind)
		if ind != "" {
			indicateurs = append(indicateurs, ind)
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating indicators: %v", err)
		return nil, fmt.Errorf("error iterating indicators: %w", err)
	}

	log.Printf("Fetched %d indicators", len(indicateurs))
	return indicateurs, nil
}
