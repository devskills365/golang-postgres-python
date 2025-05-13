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

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// pop_minute calcule une estimation d’une fréquence
func pop_minute(population int) string {
	if population == 0 {
		return "N/A"
	}
	minutesInYear := 365 * 24 * 60
	rate := float64(population) / float64(minutesInYear)

	return fmt.Sprintf("1 personne toutes les %.2f minutes", 1.0/rate)
}

// tojson convertit une structure Go en JSON pour usage dans les templates
func tojson(v interface{}) template.JS {
	a, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}
	return template.JS(a)
}

func main() {
	// Connexion à la base de données
	db, err := pgxpool.New(context.Background(), "postgres://postgres:10080805@localhost:5432/annuaire?client_encoding=UTF8")
	if err != nil {
		log.Fatalf("Erreur lors de la connexion à la base de données : %v", err)
	}
	defer db.Close()

	// Vérification de la connexion
	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("Erreur lors du ping de la base de données : %v", err)
	}

	r := gin.Default()

	// Enregistre les fonctions dans FuncMap
	r.SetFuncMap(template.FuncMap{
		"pop_minute": pop_minute,
		"tojson":     tojson,
	})

	// Fichiers statiques
	r.Static("/static", "./static")

	// Templates
	r.LoadHTMLGlob("templates/*")

	// Page d’accueil
	r.GET("/", func(c *gin.Context) {
		regions, err := handlers.GetRegions(db)
		if err != nil {
			log.Printf("Erreur lors de la récupération des régions : %v", err)
			c.HTML(http.StatusInternalServerError, "home.html", gin.H{
				"error": "Erreur lors de la récupération des régions",
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

	// Route temporaire pour /births_data
	r.GET("/births_data", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"total_births": 502150,
			"time":         time.Now().Unix(),
		})
	})

	// Route pour l'autocomplétion
	r.GET("/autocomplete", func(c *gin.Context) {
		query := strings.TrimSpace(c.Query("query"))
		if len(query) < 2 {
			c.JSON(http.StatusOK, []string{})
			return
		}

		indicateurs, err := handlers.GetIndicateurs(db)
		if err != nil {
			log.Printf("Erreur lors de la récupération des indicateurs : %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
			return
		}

		// Filtrer les indicateurs correspondant à la requête
		var matches []string
		queryLower := strings.ToLower(query)
		for _, ind := range indicateurs {
			if strings.Contains(strings.ToLower(ind), queryLower) {
				matches = append(matches, ind)
			}
		}

		c.JSON(http.StatusOK, matches)
	})

	// Route pour la recherche
	r.POST("/search", func(c *gin.Context) {
		query := strings.TrimSpace(c.PostForm("query"))
		if query == "" {
			c.HTML(http.StatusBadRequest, "home.html", gin.H{
				"error": "La requête de recherche est vide",
			})
			return
		}

		// Rediriger vers la page des résultats (temporaire)
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/search_indicators2/%s", query))
	})

	// Route temporaire pour /search_indicators2/<indicateur>
	r.GET("/search_indicators2/:indicateur", func(c *gin.Context) {
		indicateur := c.Param("indicateur")
		c.HTML(http.StatusOK, "home.html", gin.H{
			"message": fmt.Sprintf("Résultats pour l'indicateur : %s (page temporaire)", indicateur),
		})
	})

	// Lancer le serveur
	r.Run(":8080")
}
