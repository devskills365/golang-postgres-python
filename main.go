package main

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
	"version3/handlers"

	"crypto/rand"
	"encoding/hex"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pop_minute calculates an estimation of a frequency
func pop_minute(population int) string {
	if population == 0 {
		return "N/A"
	}
	minutesInYear := 365 * 24 * 60
	rate := float64(population) / float64(minutesInYear)
	return fmt.Sprintf("1 personne toutes les %.2f minutes", 1.0/rate)
}

// tojson converts a Go structure to JSON for use in templates
func tojson(v interface{}) template.JS {
	a, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(a)
}

func main() {
	// Database connection
	db, err := pgxpool.New(context.Background(), "postgres://postgres:10080805@localhost:5432/annuaire?client_encoding=UTF8")
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Verify connection
	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("Error pinging database: %v", err)
	}

	// Gin router
	r := gin.Default()

	// Register template functions
	r.SetFuncMap(template.FuncMap{
		"pop_minute": pop_minute,
		"tojson":     tojson,
	})

	// Static files
	r.Static("/static", "./static")

	// Templates
	r.LoadHTMLGlob("templates/*")

	// Session setup
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))

	// Home page
	r.GET("/", func(c *gin.Context) {
		regions, err := handlers.GetRegions(db)
		if err != nil {
			log.Printf("Error fetching regions: %v", err)
			c.HTML(http.StatusInternalServerError, "home.html", gin.H{
				"error": "Error fetching regions",
			})
			return
		}

		c.HTML(http.StatusOK, "home.html", gin.H{
			"years":                  []int{2019, 2020, 2021, 2022, 2023},
			"population":             []float64{24.0, 27.0, 27.4, 29.8, 30.38},
			"school_enrollment_rate": []int{75, 76, 78, 79, 80},
			"age_groups":             []string{"0-14 ans", "15-24 ans", "25-54 ans", "55-59 ans", "60-64", "65-69", "70-74 ans"},
			"age_distribution":       []int{40, 20, 30, 10, 18, 20},
			"regions":                regions,
			"pop_minute_val":         29389150,
			"naissance":              502150,
			"deces":                  401234,
		})
	})

	// Search indicators page
	r.GET("/structure_indicateur", func(c *gin.Context) {
		c.HTML(http.StatusOK, "search_indicateur.html", gin.H{})
	})

	// Fetch data for domains, subdomains, and indicators
	r.GET("/requete_data", func(c *gin.Context) {
		data, err := handlers.GetData(db)
		if err != nil {
			log.Printf("Error fetching data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching data"})
			return
		}
		c.JSON(http.StatusOK, data)
	})

	// Autocomplete route
	r.GET("/autocomplete", func(c *gin.Context) {
		query := strings.TrimSpace(c.Query("query"))
		if query == "" {
			c.JSON(http.StatusOK, []string{})
			return
		}
		indicateurs, err := handlers.GetAllIndicateurs(db)
		if err != nil {
			log.Printf("Error fetching indicators: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
			return
		}

		// Filter indicators matching the query
		var matches []string
		queryLower := strings.ToLower(query)
		for _, ind := range indicateurs {
			if strings.Contains(strings.ToLower(ind), queryLower) {
				matches = append(matches, ind)
			}
		}

		c.JSON(http.StatusOK, matches)
	})

	// Search route
	r.POST("/search", func(c *gin.Context) {
		query := strings.TrimSpace(c.PostForm("query"))
		if query == "" {
			c.HTML(http.StatusBadRequest, "home.html", gin.H{
				"error": "Search query is empty",
			})
			return
		}
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/requete_data/%s", query))
	})

	// Indicator results page

	// Générer un identifiant unique pour la session
	generateSessionID := func() (string, error) {
		bytes := make([]byte, 16)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}
		return hex.EncodeToString(bytes), nil
	}

	// Dans la route /requete_resultat/:indicateur
	r.GET("/requete_resultat/:indicateur", func(c *gin.Context) {
		indicateur := c.Param("indicateur")
		if indicateur == "" {
			c.HTML(http.StatusBadRequest, "no_data.html", gin.H{"error": "Aucun indicateur spécifié"})
			return
		}

		// Récupérer les données
		data, err := handlers.GetIndicateurData(db, indicateur)
		if err != nil {
			log.Printf("Error fetching indicateur data: %v", err)
			c.HTML(http.StatusInternalServerError, "no_data.html", gin.H{"error": "Erreur lors du chargement des données"})
			return
		}
		if len(data) == 0 {
			c.HTML(http.StatusOK, "no_data.html", gin.H{"error": "Aucune donnée disponible pour cet indicateur"})
			return
		}

		// Traiter les données
		processed, desaggregationColumns := handlers.ProcessIndicateurData(data)
		if len(processed) == 0 {
			c.HTML(http.StatusOK, "no_data.html", gin.H{"error": "Aucune donnée valide après traitement"})
			return
		}

		// Récupérer tous les indicateurs
		indicateurs, err := handlers.GetAllIndicateurs(db)
		if err != nil {
			log.Printf("Error fetching all indicators: %v", err)
			indicateurs = []string{}
		}

		// Convertir en JSON
		processedJSON, err := json.Marshal(processed)
		if err != nil {
			log.Printf("Error marshaling processed data: %v", err)
			c.HTML(http.StatusInternalServerError, "no_data.html", gin.H{"error": "Erreur lors du traitement des données"})
			return
		}

		// Générer un session_id unique
		sessionID, err := generateSessionID()
		if err != nil {
			log.Printf("Error generating session ID: %v", err)
			c.HTML(http.StatusInternalServerError, "no_data.html", gin.H{"error": "Erreur lors de la génération de l'identifiant"})
			return
		}

		// Stocker les données dans la table temporaire
		_, err = db.Exec(context.Background(),
			"INSERT INTO temp_processed_data (session_id, data) VALUES ($1, $2)",
			sessionID, processedJSON)
		if err != nil {
			log.Printf("Error saving to temp_processed_data: %v", err)
			c.HTML(http.StatusInternalServerError, "no_data.html", gin.H{"error": "Erreur lors de l'enregistrement des données"})
			return
		}

		// Stocker l'identifiant dans la session
		session := sessions.Default(c)
		session.Set("df_filtered_id", sessionID)
		if err := session.Save(); err != nil {
			log.Printf("Error saving session: %v", err)
			c.HTML(http.StatusInternalServerError, "no_data.html", gin.H{"error": "Erreur lors de l'enregistrement de la session"})
			return
		}

		// Préparer les données pour le template
		definition := data[0].Definition
		if definition == "" {
			definition = "Indicateur non disponible"
		}
		modeCalcul := data[0].ModeCalcul
		if modeCalcul == "" {
			modeCalcul = "Non défini"
		}

		c.HTML(http.StatusOK, "result.html", gin.H{
			"definitions":     definition,
			"mode_calcul":     modeCalcul,
			"colonne_valable": desaggregationColumns,
			"indicateurs":     indicateurs,
			"indicateur2":     indicateur,
			"df_filtered":     string(processedJSON), // Toujours envoyé au template
		})
	})

	// Process columns for pivot table
	r.POST("/process_columns", func(c *gin.Context) {
		var req handlers.PivotRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("Error binding JSON: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Données JSON invalides"})
			return
		}

		// Récupérer l'identifiant de session
		session := sessions.Default(c)
		dfFilteredID := session.Get("df_filtered_id")
		if dfFilteredID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Aucune donnée disponible dans la session"})
			return
		}

		// Récupérer les données depuis la table temporaire
		var processedJSON []byte
		err := db.QueryRow(context.Background(),
			"SELECT data FROM temp_processed_data WHERE session_id = $1", dfFilteredID).Scan(&processedJSON)
		if err != nil {
			log.Printf("Error retrieving from temp_processed_data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du chargement des données"})
			return
		}

		var processed []handlers.ProcessedData
		if err := json.Unmarshal(processedJSON, &processed); err != nil {
			log.Printf("Error unmarshaling data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du chargement des données"})
			return
		}

		// Créer le tableau croisé dynamique
		pivot, err := handlers.CreatePivotTable(processed, req)
		if err != nil {
			log.Printf("Error creating pivot table: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Erreur lors de la création du tableau croisé dynamique : %v", err)})
			return
		}

		c.JSON(http.StatusOK, pivot)
	})

	// Get filtered data
	r.GET("/get_data", func(c *gin.Context) {
		session := sessions.Default(c)
		dfFilteredJSON := session.Get("df_filtered")
		if dfFilteredJSON == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Aucune donnée filtrée disponible"})
			return
		}

		var processed []handlers.ProcessedData
		if err := json.Unmarshal([]byte(dfFilteredJSON.(string)), &processed); err != nil {
			log.Printf("Error unmarshaling session data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du chargement des données"})
			return
		}

		// Convert to records format
		records := make([]map[string]interface{}, len(processed))
		for i, row := range processed {
			records[i] = make(map[string]interface{})
			records[i]["indicateurs"] = row.Indicateur
			records[i]["valeurs"] = row.Valeur
			records[i]["annee"] = row.Annee
			records[i]["cle_pivot_table"] = row.ClePivot
			for k, v := range row.Dimensions {
				records[i][k] = v
			}
		}

		c.JSON(http.StatusOK, records)
	})

	r.GET("/get_data_temp", func(c *gin.Context) {
		sessionID := c.Query("session_id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Aucun identifiant de session fourni"})
			return
		}

		var processedJSON []byte
		err := db.QueryRow(context.Background(),
			"SELECT data FROM temp_processed_data WHERE session_id = $1", sessionID).Scan(&processedJSON)
		if err != nil {
			log.Printf("Erreur lors de la récupération des données depuis temp_processed_data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du chargement des données"})
			return
		}

		var processed []handlers.ProcessedData
		if err := json.Unmarshal(processedJSON, &processed); err != nil {
			log.Printf("Erreur lors du démarchage des données: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur lors du chargement des données"})
			return
		}

		records := make([]map[string]interface{}, len(processed))
		for i, row := range processed {
			records[i] = make(map[string]interface{})
			records[i]["indicateurs"] = row.Indicateur
			records[i]["valeurs"] = row.Valeur
			records[i]["annee"] = row.Annee
			records[i]["cle_pivot_table"] = row.ClePivot
			for k, v := range row.Dimensions {
				records[i][k] = v
			}
		}

		c.JSON(http.StatusOK, records)
	})

	// Births data
	r.GET("/births_data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"total_births": 502150,
			"time":         time.Now().Unix(),
		})
	})

	// Start server
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
