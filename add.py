
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
file_path = 'C:/Users/DELL/OneDrive - GOUVCI/DCSARD/Action_Regionale/Applications/document/importer.xlsx'

# Lire le fichier Excel
try:
    df = pd.read_excel(file_path, engine='openpyxl')
except FileNotFoundError:
    print(f"Erreur : Le fichier {file_path} n'existe pas.")
    exit(1)
except Exception as e:
    print(f"Erreur lors de la lecture du fichier Excel : {e}")
    exit(1)

# Vérifier les colonnes
expected_columns = ['indicateur_id', 'indicateur', 'definitions', 'mode_calcul']
if not all(col in df.columns for col in expected_columns):
    print(f"Erreur : Colonnes attendues : {expected_columns}, trouvées : {list(df.columns)}")
    exit(1)

# Convertir les colonnes texte en chaînes pour garantir UTF-8
df['indicateur'] = df['indicateur'].astype(str)
df['definitions'] = df['definitions'].astype(str)
df['mode_calcul'] = df['mode_calcul'].astype(str)

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
            INSERT INTO indicateurs (indicateur_id, indicateur, definitions, mode_calcul)
            VALUES (%s, %s, %s, %s)
        """)
        cursor.execute(query, (
            int(row['indicateur_id']),  # Assurer que indicateur_id est un entier
            row['indicateur'],
            row['definitions'],
            row['mode_calcul']
        ))
    conn.commit()
    print("Données importées avec succès.")
except Exception as e:
    print(f"Erreur lors de l'insertion des données : {e}")
    conn.rollback()
finally:
    cursor.close()
    conn.close()
