import pandas as pd
import psycopg2
from psycopg2 import sql

# Paramètres de connexion
db_params = {
    'dbname': 'annuaire',
    'user': 'postgres',
    'password': '10080805',
    'host': 'localhost',
    'port': '5432'
}

# Chemin du fichier Excel
file_path = "C:/Users/DELL/OneDrive - GOUVCI/DCSARD/Action_Regionale/Applications/appweb/dashbord-anstat/data_v0.xlsx"

# Lire le fichier Excel
try:
    df = pd.read_excel(file_path, engine='openpyxl')
except FileNotFoundError:
    print(f"Erreur : Le fichier {file_path} n'existe pas.")
    exit(1)
except Exception as e:
    print(f"Erreur lors de la lecture du fichier Excel : {e}")
    exit(1)

# Supprimer les espaces en trop dans les noms de colonnes
df.columns = df.columns.str.strip()

# Vérifier les colonnes
expected_columns = ['Dimension', 'Modalites', 'Indicateurs', 'Année', 'Valeurs']
if not all(col in df.columns for col in expected_columns):
    print(f"Erreur : Colonnes attendues : {expected_columns}, trouvées : {list(df.columns)}")
    exit(1)

# Convertir les colonnes texte en chaînes pour garantir UTF-8
df['Dimension'] = df['Dimension'].astype(str)
df['Modalites'] = df['Modalites'].astype(str)
df['Indicateurs'] = df['Indicateurs'].astype(str)
df['Année'] = df['Année'].astype(int)
df['Valeurs'] = df['Valeurs'].astype(float)

# Connexion à la base
try:
    conn = psycopg2.connect(**db_params)
    cursor = conn.cursor()
except Exception as e:
    print(f"Erreur de connexion à la base : {e}")
    exit(1)

# Insérer les données
try:
    for _, row in df.iterrows():
        query = sql.SQL("""
            INSERT INTO datav0 (dimension, modalites, indicateurs, annee, valeurs)
            VALUES (%s, %s, %s, %s, %s)
        """)
        cursor.execute(query, (
            row['Dimension'],
            row['Modalites'],
            row['Indicateurs'],
            row['Année'],
            row['Valeurs']
        ))
    conn.commit()
    print("Données importées avec succès.")
except Exception as e:
    print(f"Erreur lors de l'insertion des données : {e}")
    conn.rollback()
finally:
    cursor.close()
    conn.close()
