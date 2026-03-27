import os
import mysql.connector
import sys
from dotenv import load_dotenv

# Gets REMOTE server credentials NOT LOCAL
load_dotenv()
DB_HOST = os.getenv("DB_HOST")
DB_NAME = os.getenv("DB_NAME")
DB_USER = os.getenv("DB_USER")
DB_PASSWORD = os.getenv("DB_PASS")

def setup():
    print("--- Machines Got Talent: Database Setup ---")
    try:
        # 1. Initial connection to MySQL DB
        # Uses remote server credentials
        db = mysql.connector.connect(
            host = DB_HOST,
            user = DB_USER,
            password = DB_PASSWORD,
            database = DB_NAME
        )
        cursor = db.cursor()

        # 2. Create the 'prompts' table (for AI failsafe jokes)
        print("[1/3] Creating 'prompts' table...")
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS prompts (
                id INT AUTO_INCREMENT PRIMARY KEY,
                prompt TEXT NOT NULL
            )
        """)

        # 3. Create the 'users' table (for persistent stats/economy)
        print("[2/3] Creating 'users' table...")
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS users (
                id VARCHAR(255) PRIMARY KEY,
                username VARCHAR(255) UNIQUE NOT NULL,
                email VARCHAR(255) UNIQUE NOT NULL,
                password_hash VARCHAR(255) NOT NULL,
                win_count INT DEFAULT 0,
                balance INT DEFAULT 0
            )
        """)

        # 4. Seed initial "Failsafe" prompts
        print("[3/3] Seeding starter prompts...")
        starter_prompts = [
            ("Why did the AI cross the road? To optimize the path to the other side!"),
            ("A robot walks into a bar... and asks for a byte."),
            ("What do you call a comedic computer? A LOL-gorithm!"),
            ("My AI told me a joke, but it was a bit bit-ter."),
            ("Knock knock. Who's there? Java. Java who? Java nice day!"),
            ("I asked an AI to write a joke about paper. It said it was 'tear-able'."),
            ("Why was the computer cold? It left its Windows open!")
        ]

        # Only insert if the table is currently empty
        cursor.execute("SELECT COUNT(*) FROM prompts")
        if cursor.fetchone()[0] == 0:
            cursor.executemany("INSERT INTO prompts (prompt) VALUES (%s)", [(p,) for p in starter_prompts])
            print(f"      Successfully added {len(starter_prompts)} starter prompts.")
        else:
            print("      Prompts table already contains data. Skipping seeding.")

        db.commit()
        cursor.close()
        db.close()
        print(f"\nSUCCESS: Database '{DB_NAME}' is fully configured and ready for use!")

    except mysql.connector.Error as err:
        print(f"\nCRITICAL ERROR: {err}")
        print(f"\nContact Tyler to make sure the credentials are correct")
        sys.exit(1)

if __name__ == "__main__":
    setup()