import os
import mysql.connector
import sys


DB_HOST = os.getenv("MGT_DB_HOST", "localhost")
DB_USER = os.getenv("MGT_DB_USER", "root")
DB_PASSWORD = os.getenv("MGT_DB_PASSWORD", "")
DB_PORT = int(os.getenv("MGT_DB_PORT", "3306"))
DB_NAME = os.getenv("MGT_DB_NAME", "mgt_db")

def setup():
    print("--- Machines Got Talent: Database Setup ---")
    try:
        # 1. Initial connection to MySQL (without selecting a DB)
        # XAMPP Defaults: Host=localhost, User=root, Password="", Port=3306
        db = mysql.connector.connect(
            host=DB_HOST,
            user=DB_USER,
            password=DB_PASSWORD,
            port=DB_PORT,
        )
        cursor = db.cursor()

        # 2. Create the main database
        print(f"[1/4] Creating database '{DB_NAME}'...")
        cursor.execute(f"CREATE DATABASE IF NOT EXISTS `{DB_NAME}`")
        cursor.execute(f"USE `{DB_NAME}`")

        # 3. Create the 'prompts' table (for AI failsafe jokes)
        print("[2/4] Creating 'prompts' table...")
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS prompts (
                id INT AUTO_INCREMENT PRIMARY KEY,
                prompt TEXT NOT NULL
            )
        """)

        # 4. Create the 'users' table (for persistent stats/economy)
        print("[3/4] Creating 'users' table...")
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

        # 5. Seed initial "Failsafe" prompts
        print("[4/4] Seeding starter prompts...")
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
        print("\nTROUBLESHOOTING:")
        print("1. Ensure your MySQL server is running.")
        print(f"2. Confirm the connection settings are correct: host={DB_HOST} port={DB_PORT} user={DB_USER} db={DB_NAME}.")
        print("3. Set MGT_DB_HOST / MGT_DB_PORT / MGT_DB_USER / MGT_DB_PASSWORD / MGT_DB_NAME if you are not using the README defaults.")
        sys.exit(1)

if __name__ == "__main__":
    setup()
